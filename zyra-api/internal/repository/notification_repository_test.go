package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/database"
)

func TestPostgresNotificationReadIsolationAndCutoff(t *testing.T) {
	dsn := os.Getenv("ZYRA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ZYRA_TEST_DATABASE_URL to disposable PostgreSQL")
	}
	db, err := database.Connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	if err := tx.Exec("CREATE TEMP TABLE notifications (id text PRIMARY KEY, recipient_id text, actor_id text, ticket_id text, comment_id text, created_at timestamptz, read_at timestamptz, withdrawn_at timestamptz)").Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	for _, item := range []struct {
		id, recipient string
		created       time.Time
		withdrawn     *time.Time
	}{
		{"older", "recipient", now.Add(-time.Hour), nil},
		{"cutoff", "recipient", now, nil},
		{"future", "recipient", now.Add(time.Second), nil},
		{"other-user", "someone-else", now, nil},
		{"withdrawn", "recipient", now, &now},
	} {
		if err := tx.Exec("INSERT INTO notifications (id,recipient_id,created_at,withdrawn_at) VALUES (?,?,?,?)", item.id, item.recipient, item.created, item.withdrawn).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewNotificationRepository(tx)
	ctx := context.Background()
	if count, err := repo.UnreadCount(ctx, "recipient"); err != nil || count != 3 {
		t.Fatalf("initial count %d %v", count, err)
	}
	if err := repo.Read(ctx, "someone-else", "older", now); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("foreign read returned %v", err)
	}
	if err := repo.Read(ctx, "recipient", "withdrawn", now); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("withdrawn read returned %v", err)
	}
	if err := repo.Read(ctx, "recipient", "older", now); err != nil {
		t.Fatal(err)
	}
	if err := repo.Read(ctx, "recipient", "older", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var readAt time.Time
	if err := tx.Raw("SELECT read_at FROM notifications WHERE id='older'").Scan(&readAt).Error; err != nil {
		t.Fatal(err)
	}
	if !readAt.Equal(now) {
		t.Fatal("idempotent mark read overwrote first read timestamp")
	}
	if err := repo.ReadAll(ctx, "recipient", now); err != nil {
		t.Fatal(err)
	}
	if count, err := repo.UnreadCount(ctx, "recipient"); err != nil || count != 1 {
		t.Fatalf("cutoff did not preserve later unread notification: %d %v", count, err)
	}
	if count, err := repo.UnreadCount(ctx, "someone-else"); err != nil || count != 1 {
		t.Fatalf("mark-all-read affected another recipient: %d %v", count, err)
	}
	var withdrawn struct{ ReadAt *time.Time }
	if err := tx.Raw("SELECT read_at FROM notifications WHERE id='withdrawn'").Scan(&withdrawn).Error; err != nil {
		t.Fatal(err)
	}
	if withdrawn.ReadAt != nil {
		t.Fatal("mark-all-read touched withdrawn notification")
	}
}
