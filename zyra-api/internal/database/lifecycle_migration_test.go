package database_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
	"zyra-api/internal/database"
	"zyra-api/internal/models"
)

func TestPostgresLegacyLifecycleAndCommentMigration(t *testing.T) {
	dsn := os.Getenv("ZYRA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ZYRA_TEST_DATABASE_URL to disposable PostgreSQL")
	}
	root, err := database.Connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	rootPool, err := root.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer rootPool.Close()
	schema := fmt.Sprintf("zyra_legacy_test_%d", time.Now().UnixNano())
	if err := root.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer root.Exec("DROP SCHEMA " + schema + " CASCADE")
	db, err := database.Connect(dsn + " search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// These are the actual pre-lifecycle/pre-comment columns, not new models
	// with artificial default values filled in by the test.
	for _, statement := range []string{
		`CREATE TABLE users (id text PRIMARY KEY,email text NOT NULL,name text,role text,theme text,disabled boolean,password_hash text,created_at timestamptz)`,
		`CREATE TABLE ticket_events (id text PRIMARY KEY,ticket_id text,user_id text,type text,comment text,created_at timestamptz)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AutoMigrate(&models.Session{}, &models.ResetToken{}); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 9, 23, 8, 0, 0, 0, time.UTC)
	for _, fixture := range []struct {
		id       string
		disabled bool
	}{{"active-user", false}, {"retired-user", true}} {
		if err := db.Exec("INSERT INTO users VALUES (?,?,?,?,?,?,?,?)", fixture.id, fixture.id+"@example.test", fixture.id, "normal", "dark", fixture.disabled, "unchanged-password-hash", created).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&models.Session{ID: fixture.id + "-session", UserID: fixture.id, TokenHash: fixture.id + "-token", ExpiresAt: created.Add(7 * 24 * time.Hour)}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&models.ResetToken{ID: fixture.id + "-reset", UserID: fixture.id, TokenHash: fixture.id + "-reset-token", ExpiresAt: created.Add(time.Hour)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct{ id, user, kind, text string }{
		{"system-event", "", "system_findings", "System findings must stay immutable"},
		{"plain-event", "active-user", "comment", "Original comment\nSecond line"},
		{"closure-event", "retired-user", "comment_and_close", "Resolved historic issue"},
	} {
		if err := db.Exec("INSERT INTO ticket_events VALUES (?,?,?,?,?,?)", fixture.id, "ticket", fixture.user, fixture.kind, fixture.text, created).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn("users", "disabled") {
		t.Fatal("legacy disabled column remains independently editable")
	}
	var users []models.User
	if err := db.Order("id").Find(&users).Error; err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || users[0].Status != models.StatusActive || users[1].Status != models.StatusRetired {
		t.Fatalf("wrong lifecycle backfill %+v", users)
	}
	for _, user := range users {
		if user.Theme != "dark" || user.Appearance != "modern" || user.PasswordHash != "unchanged-password-hash" || user.Role != models.RoleNormal {
			t.Fatal("migration altered credentials/theme/role or missed appearance default")
		}
	}
	var sessions []models.Session
	if err := db.Order("id").Find(&sessions).Error; err != nil {
		t.Fatal(err)
	}
	if sessions[0].Revoked || !sessions[1].Revoked {
		t.Fatal("session revocation did not follow legacy disabled state")
	}
	var resets []models.ResetToken
	if err := db.Order("id").Find(&resets).Error; err != nil {
		t.Fatal(err)
	}
	if resets[0].Used || !resets[1].Used {
		t.Fatal("retired account recovery token remains usable")
	}
	var comments []models.Comment
	if err := db.Order("id").Find(&comments).Error; err != nil {
		t.Fatal(err)
	}
	if len(comments) != 2 {
		t.Fatal("system finding migrated into editable comment or user text lost")
	}
	for _, comment := range comments {
		if comment.Revision != 1 || comment.EditedAt != nil || !comment.CreatedAt.Equal(created) || comment.Content == nil {
			t.Fatalf("incorrect migrated metadata %+v", comment)
		}
		if comment.Content.Blocks[0].Children[0].Text != comment.Text {
			t.Fatal("migrated document changed legacy text")
		}
	}
	var events []models.TicketEvent
	if err := db.Order("id").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("expected original 3 events + independent closure, got %d", len(events))
	}
	var closeCount int
	for _, event := range events {
		if event.Type == "close" {
			closeCount++
			if event.UserID != "retired-user" || !event.CreatedAt.Equal(created) {
				t.Fatal("closure attribution/time changed")
			}
		}
		if event.Type == "comment" && (event.CommentID != event.ID || event.Comment != "") {
			t.Fatal("migration duplicated legacy text or lost stable comment identity")
		}
	}
	if closeCount != 1 {
		t.Fatal("immutable closure was not preserved")
	}
	before, _ := json.Marshal(events)
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	var again []models.TicketEvent
	if err := db.Order("id").Find(&again).Error; err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(again)
	if string(before) != string(after) {
		t.Fatal("second migration duplicated/reordered/changed timeline")
	}
	var count int64
	if err := db.Model(&models.Comment{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatal("second migration duplicated comments")
	}
	if err := db.Model(&models.User{}).Where("id = ?", "active-user").Update("role", "owner").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.User{}).Where("id = ?", "retired-user").Updates(map[string]any{"role": "owner", "status": "active"}).Error; err == nil {
		t.Fatal("database allowed a second owner")
	}
	if err := db.Model(&models.User{}).Where("id = ?", "active-user").Update("status", "retired").Error; err == nil {
		t.Fatal("database allowed retired owner")
	}
}
