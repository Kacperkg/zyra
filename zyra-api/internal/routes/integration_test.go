package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"zyra-api/internal/auth"
	"zyra-api/internal/database"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
	"zyra-api/internal/services"
)

func TestPostgresAPIWorkflow(t *testing.T) {
	dsn := os.Getenv("ZYRA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ZYRA_TEST_DATABASE_URL to a disposable PostgreSQL connection (keyword DSN)")
	}
	root, err := database.Connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	sqlRoot, _ := root.DB()
	defer sqlRoot.Close()
	schema := fmt.Sprintf("zyra_test_%d", time.Now().UnixNano())
	if err = root.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer root.Exec("DROP SCHEMA " + schema + " CASCADE")
	db, err := database.Connect(dsn + " search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err = database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	svc := services.New(repository.New(db), auth.New("integration-secret-at-least-32-characters"))
	if err = svc.Bootstrap(context.Background(), "admin@example.test", "a-long-test-password"); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := New(svc)
	call := func(method, path, token string, body any, want int) map[string]json.RawMessage {
		t.Helper()
		data, _ := json.Marshal(body)
		r := httptest.NewRequest(method, path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, want, w.Body.String())
		}
		out := map[string]json.RawMessage{}
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	str := func(m map[string]json.RawMessage, k string) string {
		var v string
		_ = json.Unmarshal(m[k], &v)
		return v
	}
	login := call("POST", "/api/auth/login", "", map[string]any{"email": "admin@example.test", "password": "a-long-test-password"}, 200)
	token := str(login, "access_token")
	refresh := str(login, "refresh_token")
	deadline := str(login, "session_expires_at")
	rotated := call("POST", "/api/auth/refresh", "", map[string]any{"refresh_token": refresh}, 200)
	if str(rotated, "session_expires_at") != deadline {
		t.Fatal("refresh extended session")
	}
	call("POST", "/api/auth/refresh", "", map[string]any{"refresh_token": refresh}, 401)
	token = str(rotated, "access_token")
	call("GET", "/api/clients", "", nil, 401)
	client := call("POST", "/api/clients", token, map[string]any{"name": "Example"}, 201)
	clientID := str(client, "id")
	d := call("POST", "/api/databases", token, map[string]any{"client_id": clientID, "name": "IFSPRD", "hostname": "server01", "ip": "192.0.2.1"}, 201)
	dbID := str(d, "id")
	free := 5.0
	settings := models.CheckSettings{Selected: map[string]bool{"backups": true, "tablespace": true, "missing_email": true}, Resources: map[string]map[string]models.ResourceRule{"backups": {`E:\A.DBF`: {}, `E:\B.DBF`: {}}, "tablespace": {"UNDOTBS1": {MinFreePercent: &free}}}}
	call("PUT", "/api/databases/"+dbID+"/check-settings", token, settings, 200)
	report := map[string]any{"message_id": "sample-1", "sender": "reports@example.test", "subject": "daily", "body": "ReportOn: 23-sep-2026:11:27:59, PkgVersion: 2.5\nDatabase: IFSPRD\nDatabase checks\nBackups=\\\nDatafiles needing backup:\nRMAN E:\\A.DBF 22-SEP-2026 00:42:16\nRMAN E:\\B.DBF 22-SEP-2026 00:42:16\nTablespaces=\\\nUNDOTBS1 31744 31744 30900 2.7%\nScript Info\nRun by : user@server01"}
	a := call("POST", "/api/databases/"+dbID+"/assessments", token, report, 200)
	a2 := call("POST", "/api/databases/"+dbID+"/assessments", token, report, 200)
	if str(a, "id") != str(a2, "id") {
		t.Fatal("duplicate assessment")
	}
	listed := call("GET", "/api/tickets", token, nil, 200)
	var tickets []repository.TicketSummaryRow
	_ = json.Unmarshal(listed["items"], &tickets)
	if len(tickets) != 2 {
		t.Fatalf("got %d tickets", len(tickets))
	}
	for _, ticket := range tickets {
		if ticket.Number < 1 || ticket.ClientName != "Example" || ticket.DatabaseName != "IFSPRD" {
			t.Fatalf("ticket metadata %+v", ticket)
		}
	}
	if bytes.Contains(listed["items"], []byte("A.DBF")) || bytes.Contains(listed["items"], []byte(`"evidence"`)) {
		t.Fatal("ticket list leaked evidence")
	}
	id := tickets[0].ID
	call("POST", "/api/tickets/"+id+"/close", token, map[string]any{"comment": "Resolved"}, 200)
	call("POST", "/api/tickets/"+id+"/close", token, map[string]any{}, 409)
	call("POST", "/api/tickets/"+id+"/reopen", token, map[string]any{}, 200)
	detail := call("GET", "/api/tickets/"+id, token, nil, 200)
	if _, exists := detail["events"]; exists {
		t.Fatal("ticket detail must not embed timeline events")
	}
	eventPage := call("GET", "/api/tickets/"+id+"/events", token, nil, 200)
	var events []models.TicketEvent
	_ = json.Unmarshal(eventPage["items"], &events)
	if len(events) != 3 {
		t.Fatal("missing ticket history")
	}
	if events[0].Type != "system_findings" || events[0].Comment == "" || string(eventPage["limit"]) != "50" {
		t.Fatalf("unexpected ticket timeline: %+v", events)
	}
	otherID := tickets[1].ID
	call("POST", "/api/tickets/"+otherID+"/comment-and-close", token, map[string]any{"comment": ""}, 400)
	call("POST", "/api/tickets/"+otherID+"/comment-and-close", token, map[string]any{"comment": "Investigated and resolved"}, 200)
	call("POST", "/api/tickets/"+otherID+"/comments", token, map[string]any{"comment": "Follow-up while closed"}, 200)
	closedDetail := call("GET", "/api/tickets/"+otherID, token, nil, 200)
	var closedTicket models.Ticket
	_ = json.Unmarshal(closedDetail["ticket"], &closedTicket)
	if closedTicket.Status != "closed" {
		t.Fatal("comment on closed ticket reopened it")
	}
	report["message_id"] = "sample-2"
	call("POST", "/api/databases/"+dbID+"/assessments", token, report, 200)
	sameType := call("GET", "/api/tickets?check_type="+tickets[0].CheckType+"&sort=number&order=desc", token, nil, 200)
	var sameTypeTickets []repository.TicketSummaryRow
	_ = json.Unmarshal(sameType["items"], &sameTypeTickets)
	if len(sameTypeTickets) != 2 {
		t.Fatalf("expected two tickets of the same type, got %d", len(sameTypeTickets))
	}
	newestDetail := call("GET", "/api/tickets/"+sameTypeTickets[0].ID, token, nil, 200)
	var similar []repository.SimilarTicketRow
	_ = json.Unmarshal(newestDetail["similar"], &similar)
	if len(similar) != 1 || similar[0].ID != sameTypeTickets[1].ID {
		t.Fatalf("unexpected similar issues: %+v", similar)
	}
	filtered := call("GET", "/api/tickets?client_id="+clientID+"&database_id="+dbID+"&created_from=2020-01-01&created_to=2020-01-02&sort=client&order=asc", token, nil, 200)
	if string(filtered["total"]) != "0" {
		t.Fatal("created date filter did not exclude current test tickets")
	}
	var adminUser models.User
	_ = json.Unmarshal(login["user"], &adminUser)
	history := call("GET", "/api/admin/users/"+adminUser.ID+"/closed-tickets", token, nil, 200)
	if string(history["total"]) != "2" {
		t.Fatal("reopened ticket disappeared from closure history")
	}
	user := call("POST", "/api/admin/users", token, map[string]any{"email": "normal@example.test", "name": "Normal", "password": "normal-test-password", "role": "normal"}, 201)
	normalLogin := call("POST", "/api/auth/login", "", map[string]any{"email": "normal@example.test", "password": "normal-test-password"}, 200)
	normal := str(normalLogin, "access_token")
	call("POST", "/api/clients", normal, map[string]any{"name": "denied"}, 403)
	call("PATCH", "/api/users/me", normal, map[string]any{"role": "admin"}, 403)
	call("PATCH", "/api/admin/users/"+str(user, "id"), token, map[string]any{"disabled": true}, 200)
	call("GET", "/api/users/me", normal, nil, 401)
	call("DELETE", "/api/clients/"+clientID, token, nil, 409)
	// Evaluate a closed window twice: one issue for the missing report, not two.
	call("POST", "/api/databases/"+dbID+"/email-sources", token, map[string]any{"sender": "missing@example.test", "subject": "daily", "timezone": "UTC", "enabled": true, "schedules": []map[string]string{{"name": "AM", "start": "06:00", "end": "10:00"}}}, 200)
	date := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	first := call("POST", "/api/admin/schedules/evaluate", token, map[string]any{"date": date}, 200)
	if string(first["created"]) != "1" {
		t.Fatalf("missing issue count: %s", first["created"])
	}
	again := call("POST", "/api/admin/schedules/evaluate", token, map[string]any{"date": date}, 200)
	if string(again["created"]) != "0" {
		t.Fatal("duplicate missing issue")
	}
	call("POST", "/api/auth/forgot-password", "", map[string]any{"email": "admin@example.test"}, 503)
	// The reset mechanism is exercised with an injected delivery adapter.
	sender := &captureRecovery{}
	svc.Recovery = sender
	call("POST", "/api/auth/forgot-password", "", map[string]any{"email": "admin@example.test"}, 202)
	if sender.token == "" {
		t.Fatal("no recovery token delivered")
	}
	call("POST", "/api/auth/logout", token, map[string]any{}, 200)
	call("GET", "/api/users/me", token, nil, 401)
	call("POST", "/api/auth/reset-password", "", map[string]any{"token": sender.token, "password": "a-new-long-test-password"}, 200)
	call("POST", "/api/auth/reset-password", "", map[string]any{"token": sender.token, "password": "a-new-long-test-password"}, 401)
	call("POST", "/api/auth/login", "", map[string]any{"email": "admin@example.test", "password": "a-long-test-password"}, 401)
	call("POST", "/api/auth/login", "", map[string]any{"email": "admin@example.test", "password": "a-new-long-test-password"}, 200)
	// The weak local fixture is available only while explicitly enabled, even
	// if its persisted account/session survives an API restart without the flag.
	if err := svc.BootstrapDevelopmentAdmin(context.Background()); err == nil {
		t.Fatal("development bootstrap allowed without opt-in")
	}
	svc.DevAdminLogin = true
	if err := svc.BootstrapDevelopmentAdmin(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := svc.BootstrapDevelopmentAdmin(context.Background()); err != nil {
		t.Fatal(err)
	}
	dev := call("POST", "/api/auth/login", "", map[string]any{"email": "admin", "password": "admin"}, 200)
	devAccess, devRefresh := str(dev, "access_token"), str(dev, "refresh_token")
	call("GET", "/api/admin/users", devAccess, nil, 200)
	call("POST", "/api/admin/users", devAccess, map[string]any{"email": "short@example.test", "name": "Short", "password": "admin"}, 400)
	call("POST", "/api/auth/login", "", map[string]any{"email": "admin", "password": "wrong"}, 401)
	svc.DevAdminLogin = false
	call("POST", "/api/auth/login", "", map[string]any{"email": "admin", "password": "admin"}, 401)
	call("POST", "/api/auth/login", "", map[string]any{"email": "admin@zyra.test", "password": "admin"}, 401)
	call("GET", "/api/admin/users", devAccess, nil, 401)
	call("POST", "/api/auth/refresh", "", map[string]any{"refresh_token": devRefresh}, 401)
}

type captureRecovery struct{ token string }

func (c *captureRecovery) SendReset(_ context.Context, _ string, token string) error {
	c.token = token
	return nil
}
