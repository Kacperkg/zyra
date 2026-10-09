package routes

import (
	"context"
	"testing"
	"time"
)

// Logout shares the actor lock with participating writes. Otherwise a write can
// validate a session, logout can return, and that write can still commit later.
func TestPostgresLogoutWaitsForParticipationAndRevokesRefresh(t *testing.T) {
	h := newProfileAPIHarness(t)
	login := h.login("admin", "admin")
	access := profileString(t, login, "access_token")
	refresh := profileString(t, login, "refresh_token")
	actor, sessionID, err := h.service.Authenticate(context.Background(), access)
	if err != nil {
		t.Fatal(err)
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	if err := tx.Exec("SELECT id FROM users WHERE id = ? FOR UPDATE", actor.ID).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- h.service.Logout(ctx, sessionID) }()
	select {
	case err := <-done:
		t.Fatalf("logout completed while participation still held actor lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("logout did not finish after participation released its lock")
	}
	h.call("GET", "/api/users/me", access, nil, 401)
	h.call("POST", "/api/auth/refresh", "", map[string]any{"refresh_token": refresh}, 401)
}
