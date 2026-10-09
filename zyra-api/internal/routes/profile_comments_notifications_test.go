package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
	"zyra-api/internal/auth"
	"zyra-api/internal/database"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
	"zyra-api/internal/services"
)

type profileAPIHarness struct {
	t       *testing.T
	db      *gorm.DB
	service *services.Service
	router  *gin.Engine
}

func newProfileAPIHarness(t *testing.T) *profileAPIHarness {
	t.Helper()
	dsn := os.Getenv("ZYRA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ZYRA_TEST_DATABASE_URL to a disposable PostgreSQL keyword DSN")
	}
	root, err := database.Connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	rootPool, err := root.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rootPool.Close() })
	schema := fmt.Sprintf("zyra_profile_test_%d", time.Now().UnixNano())
	if err := root.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Error(err)
		}
	})
	db, err := database.Connect(dsn + " search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	svc := services.New(repository.New(db), auth.New("profile-integration-secret-at-least-32-characters"))
	svc.DevAdminLogin = true
	if err := svc.BootstrapDevelopmentAdmin(context.Background()); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	return &profileAPIHarness{t: t, db: db, service: svc, router: New(svc)}
}

func (h *profileAPIHarness) call(method, path, token string, body any, statuses ...int) map[string]json.RawMessage {
	h.t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		h.t.Fatal(err)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(data))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, r)
	allowed := false
	for _, code := range statuses {
		if w.Code == code {
			allowed = true
		}
	}
	if !allowed {
		h.t.Fatalf("%s %s got %d, want %v: %s", method, path, w.Code, statuses, w.Body.String())
	}
	result := map[string]json.RawMessage{}
	if w.Body.Len() > 0 && w.Code != 204 {
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			h.t.Fatalf("invalid JSON: %s", w.Body.String())
		}
	}
	return result
}

func profileString(t *testing.T, value map[string]json.RawMessage, key string) string {
	t.Helper()
	var out string
	if err := json.Unmarshal(value[key], &out); err != nil {
		t.Fatalf("missing/invalid %s in %v", key, value)
	}
	return out
}

func profileObject(t *testing.T, value map[string]json.RawMessage, key string) map[string]json.RawMessage {
	t.Helper()
	var out map[string]json.RawMessage
	if err := json.Unmarshal(value[key], &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func (h *profileAPIHarness) login(email, password string) map[string]json.RawMessage {
	h.t.Helper()
	return h.call("POST", "/api/auth/login", "", map[string]any{"email": email, "password": password}, 200)
}

func (h *profileAPIHarness) createUser(owner, email, name, role string) (string, map[string]json.RawMessage) {
	h.t.Helper()
	user := h.call("POST", "/api/admin/users", owner, map[string]any{"email": email, "name": name, "role": role, "password": "a-test-password-123"}, 201)
	return profileString(h.t, user, "id"), h.login(email, "a-test-password-123")
}

func TestPostgresProfileLifecycleAndPrivateBookmarks(t *testing.T) {
	h := newProfileAPIHarness(t)
	ownerLogin := h.login("admin", "admin")
	owner := profileString(t, ownerLogin, "access_token")
	ownerUser := profileObject(t, ownerLogin, "user")
	ownerID := profileString(t, ownerUser, "id")
	if profileString(t, ownerUser, "role") != "owner" {
		t.Fatal("explicit development fixture must be owner")
	}
	adminID, adminLogin := h.createUser(owner, "admin1@example.test", "Admin One", "admin")
	admin := profileString(t, adminLogin, "access_token")
	otherAdminID, _ := h.createUser(owner, "admin2@example.test", "Admin Two", "admin")
	normalID, normalLogin := h.createUser(owner, "normal@example.test", "Normal Person", "normal")
	normal := profileString(t, normalLogin, "access_token")
	refresh := profileString(t, normalLogin, "refresh_token")
	_, strangerLogin := h.createUser(owner, "stranger@example.test", "Other Person", "normal")
	stranger := profileString(t, strangerLogin, "access_token")
	if profileString(t, profileObject(t, normalLogin, "user"), "status") != "active" {
		t.Fatal("new account must default active")
	}
	if _, exists := profileObject(t, normalLogin, "user")["disabled"]; exists {
		t.Fatal("obsolete disabled field exposed")
	}

	// Profile preferences persist through fresh authentication; URLs never cause a server fetch.
	updated := h.call("PATCH", "/api/users/me", normal, map[string]any{"avatar_url": "https://images.example.test/avatar.png", "appearance": "classic", "theme": "dark"}, 200)
	if profileString(t, updated, "appearance") != "classic" {
		t.Fatal("appearance not saved")
	}
	for _, invalidURL := range []string{"http://images.example.test/a.png", "javascript:alert(1)", "https://user:pass@example.test/a.png", "https://"} {
		h.call("PATCH", "/api/users/me", normal, map[string]any{"avatar_url": invalidURL}, 400)
	}
	h.call("PATCH", "/api/users/me", normal, map[string]any{"appearance": "surprise"}, 400)
	h.call("PATCH", "/api/users/me", normal, map[string]any{"role": "owner"}, 403)
	h.call("PATCH", "/api/admin/users/"+normalID, owner, map[string]any{"disabled": true}, 400)
	newLogin := h.login("normal@example.test", "a-test-password-123")
	profile := profileObject(t, newLogin, "user")
	if profileString(t, profile, "appearance") != "classic" || profileString(t, profile, "theme") != "dark" {
		t.Fatal("preferences not retained on fresh login")
	}
	h.call("PATCH", "/api/users/me", normal, map[string]any{"avatar_url": ""}, 200)

	// Ticket fixture goes through the real parser and API so bookmark access is authentic.
	client := h.call("POST", "/api/clients", owner, map[string]any{"name": "QA Customer"}, 201)
	db := h.call("POST", "/api/databases", owner, map[string]any{"client_id": profileString(t, client, "id"), "name": "QA_DB", "hostname": "qa-host"}, 201)
	dbID := profileString(t, db, "id")
	h.call("PUT", "/api/databases/"+dbID+"/check-settings", owner, map[string]any{"selected": map[string]bool{"missing_email": true}}, 200)
	for i := 0; i < 2; i++ {
		h.call("POST", "/api/databases/"+dbID+"/assessments", owner, map[string]any{"message_id": fmt.Sprintf("profile-fixture-%d", i), "sender": "reports@example.test", "subject": "daily", "body": "Incomplete QA report"}, 200)
	}
	list := h.call("GET", "/api/tickets", owner, nil, 200)
	var tickets []map[string]json.RawMessage
	if err := json.Unmarshal(list["items"], &tickets); err != nil || len(tickets) != 2 {
		t.Fatalf("expected two fixture tickets: %s", list["items"])
	}
	ticketID := profileString(t, tickets[0], "id")
	otherTicketID := profileString(t, tickets[1], "id")
	h.call("PUT", "/api/users/me/saved-tickets/"+ticketID, normal, map[string]any{"title": "My private fix"}, 200, 201)
	h.call("PUT", "/api/users/me/saved-tickets/"+ticketID, normal, map[string]any{"title": "forged", "user_id": ownerID}, 400)
	h.call("PUT", "/api/users/me/saved-tickets/"+otherTicketID, normal, map[string]any{"title": "Newer saved item"}, 200, 201)
	before := h.call("GET", "/api/users/me/saved-tickets", normal, nil, 200)
	if string(before["total"]) != "2" {
		t.Fatal("bookmark saves missing")
	}
	h.call("PUT", "/api/users/me/saved-tickets/"+ticketID, normal, map[string]any{"title": "Retitled private fix"}, 200, 201)
	after := h.call("GET", "/api/users/me/saved-tickets", normal, nil, 200)
	var beforeRows, afterRows []map[string]json.RawMessage
	_ = json.Unmarshal(before["items"], &beforeRows)
	_ = json.Unmarshal(after["items"], &afterRows)
	if string(after["total"]) != "2" || len(afterRows) != 2 {
		t.Fatal("idempotent save created duplicates")
	}
	if profileString(t, beforeRows[0], "id") != profileString(t, afterRows[0], "id") {
		t.Fatal("retitling changed saved ordering")
	}
	for _, token := range []string{stranger, admin, owner} {
		private := h.call("GET", "/api/users/me/saved-tickets", token, nil, 200)
		if string(private["total"]) != "0" {
			t.Fatal("another account can see private bookmarks")
		}
	}
	searched := h.call("GET", "/api/users/me/saved-tickets?q=Retitled", normal, nil, 200)
	if string(searched["total"]) != "1" {
		t.Fatal("personal-title search did not match")
	}
	h.call("DELETE", "/api/users/me/saved-tickets/"+ticketID, stranger, nil, 200, 204)
	if string(h.call("GET", "/api/users/me/saved-tickets", normal, nil, 200)["total"]) != "2" {
		t.Fatal("foreign unsave changed owner bookmark")
	}

	// Boundaries are evaluated on the original target, not a demotion in the same body.
	h.call("PATCH", "/api/admin/users/"+otherAdminID, admin, map[string]any{"role": "normal", "status": "retired"}, 403)
	h.call("PATCH", "/api/admin/users/"+adminID, admin, map[string]any{"status": "retired"}, 400, 403)
	h.call("PATCH", "/api/admin/users/"+ownerID, owner, map[string]any{"status": "retired"}, 403)
	h.call("PATCH", "/api/admin/users/"+ownerID, owner, map[string]any{"role": "admin"}, 403)
	h.call("POST", "/api/admin/users", owner, map[string]any{"email": "owner2@example.test", "name": "Second Owner", "password": "a-test-password-123", "role": "owner"}, 400, 403, 409)
	h.call("PATCH", "/api/admin/users/"+normalID, admin, map[string]any{"status": "retired"}, 200)
	h.call("GET", "/api/users/me", normal, nil, 401)
	h.call("POST", "/api/auth/refresh", "", map[string]any{"refresh_token": refresh}, 401)
	h.call("POST", "/api/auth/login", "", map[string]any{"email": "normal@example.test", "password": "a-test-password-123"}, 401)
	h.call("PATCH", "/api/admin/users/"+normalID, admin, map[string]any{"status": "active"}, 200)
	h.call("PATCH", "/api/admin/users/"+normalID, admin, map[string]any{"status": "active"}, 200)
	var audit []models.AccountLifecycle
	if err := h.db.Where("target_id = ?", normalID).Order("created_at ASC, id ASC").Find(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if len(audit) != 2 {
		t.Fatalf("expected retirement/reactivation audit only, got %d", len(audit))
	}
	if audit[0].ActorID != adminID || audit[0].PreviousStatus != models.StatusActive || audit[0].Status != models.StatusRetired || audit[1].ActorID != adminID || audit[1].PreviousStatus != models.StatusRetired || audit[1].Status != models.StatusActive {
		t.Fatalf("wrong lifecycle audit: %+v", audit)
	}
	for _, entry := range audit {
		if entry.TargetID != normalID || entry.Role != models.RoleNormal || entry.PreviousRole != models.RoleNormal || entry.CreatedAt.IsZero() {
			t.Fatal("incomplete lifecycle audit")
		}
	}
	h.call("GET", "/api/users/me", normal, nil, 401)
	h.call("POST", "/api/auth/refresh", "", map[string]any{"refresh_token": refresh}, 401)
	restored := profileString(t, h.login("normal@example.test", "a-test-password-123"), "access_token")
	if string(h.call("GET", "/api/users/me/saved-tickets", restored, nil, 200)["total"]) != "2" {
		t.Fatal("retirement/reactivation lost bookmarks")
	}
	h.call("DELETE", "/api/users/me/saved-tickets/"+ticketID, restored, nil, 200, 204)
	h.call("DELETE", "/api/users/me/saved-tickets/"+ticketID, restored, nil, 200, 204)
	if strings.Contains(string(h.call("GET", "/api/users/me", restored, nil, 200)["avatar_url"]), "https://") {
		t.Fatal("cleared avatar restored unexpectedly")
	}
}

func TestPostgresRichCommentsMentionsAndOwnership(t *testing.T) {
	h := newProfileAPIHarness(t)
	owner := profileString(t, h.login("admin", "admin"), "access_token")
	authorID, authorLogin := h.createUser(owner, "author@example.test", "Comment Author", "normal")
	author := profileString(t, authorLogin, "access_token")
	recipientID, recipientLogin := h.createUser(owner, "recipient@example.test", "Mention Recipient", "trusted")
	recipient := profileString(t, recipientLogin, "access_token")
	otherID, otherLogin := h.createUser(owner, "other@example.test", "Second Recipient", "normal")
	other := profileString(t, otherLogin, "access_token")
	client := h.call("POST", "/api/clients", owner, map[string]any{"name": "Comment QA"}, 201)
	db := h.call("POST", "/api/databases", owner, map[string]any{"client_id": profileString(t, client, "id"), "name": "COMMENTS", "hostname": "qa-host"}, 201)
	dbID := profileString(t, db, "id")
	h.call("PUT", "/api/databases/"+dbID+"/check-settings", owner, map[string]any{"selected": map[string]bool{"missing_email": true}}, 200)
	h.call("POST", "/api/databases/"+dbID+"/assessments", owner, map[string]any{"message_id": "comment-fixture", "sender": "reports@example.test", "subject": "daily", "body": "Incomplete QA report"}, 200)
	listed := h.call("GET", "/api/tickets", owner, nil, 200)
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(listed["items"], &rows); err != nil || len(rows) != 1 {
		t.Fatalf("ticket fixture missing %s", listed["items"])
	}
	ticketID := profileString(t, rows[0], "id")
	path := "/api/tickets/" + ticketID
	document := func(text string, recipients ...string) map[string]any {
		children := []map[string]any{{"type": "text", "text": text, "bold": true}}
		for _, id := range recipients {
			children = append(children, map[string]any{"type": "mention", "user_id": id})
		}
		return map[string]any{"schema_version": 1, "content": map[string]any{"blocks": []map[string]any{{"type": "paragraph", "children": children}}}}
	}
	items := func(result map[string]json.RawMessage) []map[string]json.RawMessage {
		t.Helper()
		var out []map[string]json.RawMessage
		if err := json.Unmarshal(result["items"], &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	timeline := func() []map[string]json.RawMessage { return items(h.call("GET", path+"/events", author, nil, 200)) }
	notifications := func(token string) []map[string]json.RawMessage {
		return items(h.call("GET", "/api/users/me/notifications", token, nil, 200))
	}
	assertCount := func(token string, want int) {
		t.Helper()
		response := h.call("GET", "/api/users/me/notifications/unread-count", token, nil, 200)
		if string(response["count"]) != fmt.Sprint(want) {
			t.Fatalf("unread count got %s want %d", response["count"], want)
		}
	}

	// Bounded discovery exposes identity only and excludes retired accounts.
	suggestions := h.call("GET", "/api/users/mention-options?q=Recipient&limit=1", author, nil, 200)
	if len(items(suggestions)) != 1 {
		t.Fatal("mention result limit ignored")
	}
	for _, suggestion := range items(suggestions) {
		for _, field := range []string{"email", "role", "password_hash", "status"} {
			if _, ok := suggestion[field]; ok {
				t.Fatalf("mention response exposed %s", field)
			}
		}
	}
	if len(items(h.call("GET", "/api/users/mention-options?q=R", author, nil, 200))) != 0 {
		t.Fatal("short mention query returned directory entries")
	}
	h.call("GET", "/api/users/mention-options?q=Recipient&limit=21", author, nil, 400)

	// The server validates meaningful content, protocols and mutually exclusive compatibility envelopes.
	h.call("POST", path+"/comment-and-close", author, document("   "), 400)
	unsafe := document("unsafe")
	unsafe["content"] = map[string]any{"blocks": []map[string]any{{"type": "paragraph", "children": []map[string]any{{"type": "text", "text": "click", "href": "javascript:alert(1)"}}}}}
	h.call("POST", path+"/comments", author, unsafe, 400)
	conflicting := document("new")
	conflicting["comment"] = "legacy"
	h.call("POST", path+"/comments", author, conflicting, 400)
	h.call("POST", path+"/comments", author, document("forged", "nonexistent-user"), 400, 404)
	if len(timeline()) != 1 {
		t.Fatal("rejected comment left a visible event")
	}
	open := h.call("GET", path, author, nil, 200)
	if profileString(t, profileObject(t, open, "ticket"), "status") != "open" {
		t.Fatal("invalid comment-and-close changed status")
	}
	initialSystem := profileString(t, timeline()[0], "id")
	systemEdit := document("overwrite")
	systemEdit["revision"] = 1
	h.call("PATCH", path+"/comments/"+initialSystem, owner, systemEdit, 403, 404)

	// Repeated mentions deduplicate, self-mentions do not notify, and rich text derives its preview.
	h.call("POST", path+"/comments", author, document("Investigated the tablespace. ", recipientID, recipientID, authorID), 200, 201)
	events := timeline()
	if len(events) != 2 {
		t.Fatalf("unexpected event count %d", len(events))
	}
	event := events[1]
	var mentionUsers []map[string]json.RawMessage
	if err := json.Unmarshal(event["mention_users"], &mentionUsers); err != nil || len(mentionUsers) != 2 {
		t.Fatal("mention display identities missing or duplicated", err)
	}
	for _, user := range mentionUsers {
		if _, ok := user["email"]; ok {
			t.Fatal("mention display identity exposed email")
		}
		if profileString(t, user, "id") == recipientID && profileString(t, user, "name") != "Mention Recipient" {
			t.Fatal("mention display name incorrect")
		}
	}
	commentID := profileString(t, event, "comment_id")
	createdAt := profileString(t, event, "created_at")
	if string(event["revision"]) != "1" || !bytes.Contains(event["comment"], []byte("Investigated the tablespace.")) {
		t.Fatal("rich comment metadata/derived text missing")
	}
	assertCount(recipient, 1)
	assertCount(author, 0)
	assertCount(other, 0)
	firstNotice := notifications(recipient)
	if len(firstNotice) != 1 {
		t.Fatal("duplicate mentions produced duplicate notifications")
	}
	noticeID := profileString(t, firstNotice[0], "id")
	if profileString(t, firstNotice[0], "ticket_id") != ticketID || profileString(t, firstNotice[0], "comment_id") != commentID {
		t.Fatal("notification target metadata incorrect")
	}
	h.call("PATCH", "/api/users/me/notifications/"+noticeID+"/read", owner, nil, 404)
	h.call("PATCH", "/api/users/me/notifications/"+noticeID+"/read", other, nil, 404)
	h.call("PATCH", "/api/users/me/notifications/"+noticeID+"/read", recipient, nil, 200)
	h.call("PATCH", "/api/users/me/notifications/"+noticeID+"/read", recipient, nil, 200)
	assertCount(recipient, 0)
	if len(notifications(recipient)) != 1 {
		t.Fatal("reading removed notification history")
	}
	edit := document("Edited and adding another person. ", recipientID, otherID)
	edit["revision"] = 1
	h.call("PATCH", path+"/comments/"+commentID, owner, edit, 403)
	h.call("DELETE", path+"/comments/"+commentID+"?revision=1", owner, nil, 403)
	h.call("PATCH", path+"/comments/"+commentID, author, edit, 200)
	h.call("PATCH", path+"/comments/"+commentID, author, edit, 409)
	h.call("DELETE", path+"/comments/"+commentID+"?revision=1", author, nil, 409)
	assertCount(other, 1)
	assertCount(recipient, 0)
	event = timeline()[1]
	if profileString(t, event, "created_at") != createdAt || string(event["revision"]) != "2" || string(event["edited_at"]) == "null" || len(event["edited_at"]) == 0 {
		t.Fatal("edit changed chronological position or failed to mark edited")
	}
	remove := document("Removed mention.", otherID)
	remove["revision"] = 2
	h.call("PATCH", path+"/comments/"+commentID, author, remove, 200)
	if len(notifications(recipient)) != 0 {
		t.Fatal("removed mention did not withdraw read notification")
	}
	h.call("PATCH", "/api/users/me/notifications/"+noticeID+"/read", recipient, nil, 404)
	readd := document("Readded mention.", recipientID, otherID)
	readd["revision"] = 3
	h.call("PATCH", path+"/comments/"+commentID, author, readd, 200)
	assertCount(recipient, 0)
	if len(notifications(recipient)) != 0 {
		t.Fatal("remove/readd revived once-ever notification")
	}
	h.call("DELETE", path+"/comments/"+commentID+"?revision=4", author, nil, 200, 204)
	if len(timeline()) != 1 {
		t.Fatal("deleted comment remained in timeline or placeholder added")
	}
	var retained struct {
		Text    string
		Content json.RawMessage
	}
	if err := h.db.Table("comments").Select("text, content").Where("id = ?", commentID).Scan(&retained).Error; err != nil {
		t.Fatal(err)
	}
	if retained.Text != "" || (len(retained.Content) > 0 && string(retained.Content) != "null") {
		t.Fatal("deleted rich comment content was retained")
	}
	var retainedEvent string
	if err := h.db.Table("ticket_events").Select("comment").Where("id = ?", commentID).Scan(&retainedEvent).Error; err != nil {
		t.Fatal(err)
	}
	if retainedEvent != "" {
		t.Fatal("deleted comment text remained on legacy timeline event")
	}
	assertCount(other, 0)
	if len(notifications(other)) != 0 {
		t.Fatal("comment deletion did not withdraw all notifications")
	}

	// Closing remains immutable even after deleting its independently editable comment.
	h.call("POST", path+"/comment-and-close", author, document("Resolved.", otherID), 200, 201)
	events = timeline()
	var closingCommentID string
	for _, item := range events {
		if len(item["comment_id"]) > 0 && string(item["comment_id"]) != "\"\"" {
			closingCommentID = profileString(t, item, "comment_id")
		}
	}
	if closingCommentID == "" {
		t.Fatal("closure comment identity missing")
	}
	h.call("DELETE", path+"/comments/"+closingCommentID+"?revision=1", author, nil, 200, 204)
	closed := h.call("GET", path, author, nil, 200)
	if profileString(t, profileObject(t, closed, "ticket"), "status") != "closed" {
		t.Fatal("deleting closure comment reopened ticket")
	}
	var closeEvents int
	for _, item := range timeline() {
		if profileString(t, item, "type") == "close" {
			closeEvents++
		}
	}
	if closeEvents != 1 {
		t.Fatal("closure status evidence disappeared")
	}
	history := h.call("GET", "/api/admin/users/"+authorID+"/closed-tickets", owner, nil, 200)
	if string(history["total"]) != "1" {
		t.Fatal("closure reporting lost deleted-comment closure")
	}

	// Retirement affects future eligibility without withdrawing a recipient's past notification.
	h.call("POST", path+"/comments", author, document("Before retirement.", recipientID), 200, 201)
	beforeRetirement := timeline()
	historicalCommentID := profileString(t, beforeRetirement[len(beforeRetirement)-1], "comment_id")
	assertCount(recipient, 1)
	h.call("PATCH", "/api/admin/users/"+recipientID, owner, map[string]any{"status": "retired"}, 200)
	for _, item := range items(h.call("GET", "/api/users/mention-options?q=Mention", author, nil, 200)) {
		if profileString(t, item, "id") == recipientID {
			t.Fatal("retired recipient remained in suggestions")
		}
	}
	h.call("POST", path+"/comments", author, document("Stale picker selection.", recipientID), 400, 403)
	preserve := document("Edited while preserving historical mention.", recipientID)
	preserve["revision"] = 1
	h.call("PATCH", path+"/comments/"+historicalCommentID, author, preserve, 200)
	h.call("PATCH", "/api/admin/users/"+recipientID, owner, map[string]any{"status": "active"}, 200)
	reactivated := profileString(t, h.login("recipient@example.test", "a-test-password-123"), "access_token")
	assertCount(reactivated, 1)
	h.call("POST", "/api/users/me/notifications/read-all", reactivated, nil, 200)
	assertCount(reactivated, 0)

	// Two editors holding revision 1 cannot silently overwrite each other.
	h.call("POST", path+"/comments", author, document("Concurrent revision fixture"), 200, 201)
	events = timeline()
	concurrentID := profileString(t, events[len(events)-1], "comment_id")
	codes := make(chan int, 2)
	for _, text := range []string{"First editor", "Second editor"} {
		payload := document(text)
		payload["revision"] = 1
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		go func(body []byte) {
			request := httptest.NewRequest("PATCH", path+"/comments/"+concurrentID, bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+author)
			response := httptest.NewRecorder()
			h.router.ServeHTTP(response, request)
			codes <- response.Code
		}(encoded)
	}
	first, second := <-codes, <-codes
	if !((first == 200 && second == 409) || (first == 409 && second == 200)) {
		t.Fatalf("concurrent revision outcomes %d/%d", first, second)
	}
}

func TestPostgresExplicitOwnerBootstrapPreservesFixture(t *testing.T) {
	h := newProfileAPIHarness(t)
	var before struct {
		ID           string
		PasswordHash string
		Role         string
	}
	if err := h.db.Table("users").Where("email = ?", "admin@zyra.test").First(&before).Error; err != nil {
		t.Fatal(err)
	}
	if before.Role != "owner" {
		t.Fatal("development fixture owner missing")
	}
	if err := h.service.BootstrapDevelopmentAdmin(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := h.service.EnsureOwner(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if err := h.service.EnsureOwner(context.Background(), "different@example.test"); err == nil {
		t.Fatal("mismatched explicit owner selection accepted")
	}
	var after struct {
		ID           string
		PasswordHash string
		Role         string
	}
	if err := h.db.Table("users").Where("email = ?", "admin@zyra.test").First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("repeat bootstrap changed fixture identity, password or owner")
	}
	var owners int64
	if err := h.db.Table("users").Where("role = 'owner'").Count(&owners).Error; err != nil {
		t.Fatal(err)
	}
	if owners != 1 {
		t.Fatalf("owner count %d", owners)
	}
}

func TestPostgresRetirementBlocksStaleInFlightActor(t *testing.T) {
	h := newProfileAPIHarness(t)
	owner := profileString(t, h.login("admin", "admin"), "access_token")
	id, login := h.createUser(owner, "queued@example.test", "Original Name", "normal")
	var actor models.User
	if err := json.Unmarshal(login["user"], &actor); err != nil {
		t.Fatal(err)
	}
	tx := h.db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	if err := tx.Exec("SELECT id FROM users WHERE id = ? FOR UPDATE", id).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	newName := "Must not commit"
	go func() { _, err := h.service.SaveUser(ctx, actor, id, services.UserInput{Name: &newName}); done <- err }()
	if err := tx.Exec("UPDATE users SET status='retired' WHERE id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("stale active actor mutated profile after retirement committed")
		}
	case <-ctx.Done():
		t.Fatal("queued mutation deadlocked")
	}
	var name string
	if err := h.db.Table("users").Select("name").Where("id = ?", id).Scan(&name).Error; err != nil {
		t.Fatal(err)
	}
	if name != "Original Name" {
		t.Fatalf("in-flight mutation persisted: %q", name)
	}
}
