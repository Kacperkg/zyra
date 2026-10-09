package routes

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
	"zyra-api/internal/services"
)

// A request can authenticate before retirement, pause, then resume after
// reactivation. Its old context must remain revoked inside every transaction.
func TestPostgresStaleSessionCannotMutateAfterReactivation(t *testing.T) {
	h := newProfileAPIHarness(t)
	owner := profileString(t, h.login("admin", "admin"), "access_token")
	authorID, login := h.createUser(owner, "stale@example.test", "Stale Actor", "normal")
	token := profileString(t, login, "access_token")
	actor, sid, err := h.service.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	stale := services.WithSessionContext(context.Background(), sid)
	client := h.call("POST", "/api/clients", owner, map[string]any{"name": "Stale Customer"}, 201)
	db := h.call("POST", "/api/databases", owner, map[string]any{"client_id": profileString(t, client, "id"), "name": "STALE_DB"}, 201)
	dbID := profileString(t, db, "id")
	h.call("PUT", "/api/databases/"+dbID+"/check-settings", owner, map[string]any{"selected": map[string]bool{"missing_email": true}}, 200)
	h.call("POST", "/api/databases/"+dbID+"/assessments", owner, map[string]any{"message_id": "stale-fixture", "sender": "reports@example.test", "subject": "daily", "body": "Incomplete report"}, 200)
	var tickets []models.Ticket
	if err := json.Unmarshal(h.call("GET", "/api/tickets", owner, nil, 200)["items"], &tickets); err != nil || len(tickets) != 1 {
		t.Fatal("ticket fixture missing", err)
	}
	ticketID := tickets[0].ID
	text := "Original content"
	if _, err := h.service.WriteComment(stale, actor, ticketID, "", services.CommentInput{Comment: &text}, false); err != nil {
		t.Fatal(err)
	}
	var comments []models.Comment
	if err := h.db.Where("ticket_id = ?", ticketID).Find(&comments).Error; err != nil || len(comments) != 1 {
		t.Fatal("comment fixture missing", err)
	}
	commentID := comments[0].ID
	h.call("PATCH", "/api/admin/users/"+authorID, owner, map[string]any{"status": "retired"}, 200)
	h.call("PATCH", "/api/admin/users/"+authorID, owner, map[string]any{"status": "active"}, 200)
	assertUnauthorized := func(err error) {
		t.Helper()
		if !errors.Is(err, apperrors.ErrUnauthorized) {
			t.Fatalf("revoked in-flight session returned %v", err)
		}
	}
	theme := "dark"
	_, err = h.service.SaveUser(stale, actor, actor.ID, services.UserInput{Theme: &theme})
	assertUnauthorized(err)
	_, err = h.service.WriteComment(stale, actor, ticketID, "", services.CommentInput{Comment: &text}, false)
	assertUnauthorized(err)
	_, err = h.service.WriteComment(stale, actor, ticketID, commentID, services.CommentInput{Comment: &text, Revision: 1}, false)
	assertUnauthorized(err)
	assertUnauthorized(h.service.DeleteComment(stale, actor, ticketID, commentID, 1))
	_, err = h.service.TicketAction(stale, actor, ticketID, "close", "")
	assertUnauthorized(err)
	h.call("POST", "/api/tickets/"+ticketID+"/close", owner, nil, 200)
	_, err = h.service.TicketAction(stale, actor, ticketID, "reopen", "")
	assertUnauthorized(err)
	var persisted models.Comment
	if err := h.db.First(&persisted, "id = ?", commentID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Revision != 1 || persisted.DeletedAt != nil || persisted.Text != text {
		t.Fatal("revoked request mutated original comment")
	}
}
