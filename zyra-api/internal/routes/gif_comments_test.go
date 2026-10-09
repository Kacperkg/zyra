package routes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPostgresGIFCommentsLifecycle(t *testing.T) {
	h := newProfileAPIHarness(t)
	owner := profileString(t, h.login("admin", "admin"), "access_token")
	_, login := h.createUser(owner, "gif-author@example.test", "GIF Author", "normal")
	author := profileString(t, login, "access_token")
	recipientID, recipientLogin := h.createUser(owner, "gif-recipient@example.test", "GIF Recipient", "normal")
	recipient := profileString(t, recipientLogin, "access_token")
	client := h.call("POST", "/api/clients", owner, map[string]any{"name": "GIF Customer"}, 201)
	db := h.call("POST", "/api/databases", owner, map[string]any{"client_id": profileString(t, client, "id"), "name": "GIF_DB"}, 201)
	dbID := profileString(t, db, "id")
	h.call("PUT", "/api/databases/"+dbID+"/check-settings", owner, map[string]any{"selected": map[string]bool{"missing_email": true}}, 200)
	h.call("POST", "/api/databases/"+dbID+"/assessments", owner, map[string]any{"message_id": "gif-fixture", "sender": "reports@example.test", "subject": "daily", "body": "Incomplete report"}, 200)
	rows := func(result map[string]json.RawMessage) []map[string]json.RawMessage {
		t.Helper()
		var out []map[string]json.RawMessage
		if err := json.Unmarshal(result["items"], &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	ticketID := profileString(t, rows(h.call("GET", "/api/tickets", owner, nil, 200))[0], "id")
	path := "/api/tickets/" + ticketID
	doc := func(src, alt string, mentions bool) map[string]any {
		children := []map[string]any{{"type": "gif", "src": src, "alt": alt}}
		if mentions {
			children = append(children, map[string]any{"type": "mention", "user_id": recipientID}, map[string]any{"type": "mention", "user_id": recipientID})
		}
		return map[string]any{"schema_version": 1, "content": map[string]any{"blocks": []map[string]any{{"type": "paragraph", "children": children}}}}
	}
	for _, src := range []string{"http://example.test/a.gif", "data:image/gif;base64,AAAA", "https://user:pass@example.test/a.gif"} {
		h.call("POST", path+"/comment-and-close", author, doc(src, "", false), 400)
	}
	if len(rows(h.call("GET", path+"/events", author, nil, 200))) != 1 {
		t.Fatal("invalid GIF persisted a timeline event")
	}
	h.call("POST", path+"/comment-and-close", author, doc("https://images.example.test/animation?id=1", "Backup fixed", false), 200, 201)
	events := rows(h.call("GET", path+"/events", author, nil, 200))
	var commentID, created string
	for _, event := range events {
		if string(event["type"]) == "\"comment\"" {
			commentID = profileString(t, event, "comment_id")
			created = profileString(t, event, "created_at")
			if profileString(t, event, "comment") != "[GIF: Backup fixed]" || !strings.Contains(string(event["content"]), "https://images.example.test/animation?id=1") {
				t.Fatal("GIF source/projection not persisted")
			}
		}
	}
	if commentID == "" {
		t.Fatal("GIF-only closure comment missing")
	}
	edit := doc("https://images.example.test/new.gif", "@fake", true)
	edit["revision"] = 1
	h.call("PATCH", path+"/comments/"+commentID, author, edit, 200)
	edit["revision"] = 2
	h.call("PATCH", path+"/comments/"+commentID, author, edit, 200)
	if string(h.call("GET", "/api/users/me/notifications/unread-count", recipient, nil, 200)["count"]) != "1" {
		t.Fatal("GIF edits duplicated or omitted mention notification")
	}
	for _, event := range rows(h.call("GET", path+"/events", author, nil, 200)) {
		if string(event["comment_id"]) == "\""+commentID+"\"" && profileString(t, event, "created_at") != created {
			t.Fatal("GIF edit changed timeline order")
		}
	}
	h.call("DELETE", path+"/comments/"+commentID+"?revision=3", author, nil, 200, 204)
	var retained struct {
		Text    string
		Content json.RawMessage
	}
	if err := h.db.Table("comments").Select("text,content").Where("id = ?", commentID).Scan(&retained).Error; err != nil {
		t.Fatal(err)
	}
	if retained.Text != "" || (len(retained.Content) > 0 && string(retained.Content) != "null") {
		t.Fatal("deleted GIF URL/content retained")
	}
	if string(h.call("GET", "/api/users/me/notifications/unread-count", recipient, nil, 200)["count"]) != "0" {
		t.Fatal("deleted GIF comment did not withdraw mention")
	}
	if profileString(t, profileObject(t, h.call("GET", path, author, nil, 200), "ticket"), "status") != "closed" {
		t.Fatal("deleting GIF comment removed closure")
	}
}
