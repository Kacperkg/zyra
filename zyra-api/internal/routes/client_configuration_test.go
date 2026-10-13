package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/gorm/logger"
	"testing"
	"time"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type configurationQueryCounter struct {
	logger.Interface
	count int
}

func (c *configurationQueryCounter) Trace(ctx context.Context, start time.Time, query func() (string, int64), err error) {
	c.count++
	c.Interface.Trace(ctx, start, query, err)
}

func TestPostgresClientConfigurationAndAssessmentSummaries(t *testing.T) {
	h := newProfileAPIHarness(t)
	owner := profileString(t, h.login("admin", "admin"), "access_token")
	client := h.call("POST", "/api/clients", owner, map[string]any{"name": "Configured client"}, 201)
	clientID := profileString(t, client, "id")
	database := h.call("POST", "/api/databases", owner, map[string]any{"client_id": clientID, "name": "CONFIG_DB"}, 201)
	databaseID := profileString(t, database, "id")
	if profileString(t, database, "engine") != "oracle" || string(database["max_lag_sequences"]) != "null" {
		t.Fatal("unsafe defaults")
	}
	for _, value := range []any{-1, 1.5, 2147483648, "5"} {
		h.call("PATCH", "/api/databases/"+databaseID, owner, map[string]any{"max_lag_sequences": value}, 400)
	}
	zero := h.call("PATCH", "/api/databases/"+databaseID, owner, map[string]any{"max_lag_sequences": 0}, 200)
	if string(zero["max_lag_sequences"]) != "0" {
		t.Fatal("zero lost")
	}
	retained := h.call("PATCH", "/api/databases/"+databaseID, owner, map[string]any{"notes": "keep lag"}, 200)
	if string(retained["max_lag_sequences"]) != "0" {
		t.Fatal("partial update lost lag")
	}
	h.call("PATCH", "/api/databases/"+databaseID, owner, map[string]any{"max_lag_sequences": nil}, 200)
	h.call("PATCH", "/api/databases/"+databaseID, owner, map[string]any{"engine": "sql"}, 400)
	sql := h.call("POST", "/api/databases", owner, map[string]any{"client_id": clientID, "name": "SQL_METADATA", "engine": "sql"}, 201)
	h.call("POST", "/api/databases/"+profileString(t, sql, "id")+"/assessments", owner, map[string]any{"message_id": "sql-mail", "body": "SQL body"}, 400)
	filtered := h.call("GET", "/api/databases?client_id="+clientID+"&engine=oracle", owner, nil, 200)
	if string(filtered["total"]) != "1" {
		t.Fatal("engine filtering")
	}
	h.call("PUT", "/api/databases/"+databaseID+"/check-settings", owner, map[string]any{"selected": map[string]bool{"backups": true}}, 200)
	base := time.Now().UTC()
	var firstID string
	for i, identity := range []string{"observed-new", "observed-old", ""} {
		when := base.Add(-time.Duration(i) * time.Hour)
		body := "ReportOn: 13-oct-2026:06:00:00\nUniqueID: " + identity + "\nDatabase: CONFIG_DB\nDatabase checks\nBackups=OK!\nScript Info\nRun by : oracle@host"
		a := h.call("POST", "/api/databases/"+databaseID+"/assessments", owner, map[string]any{"message_id": fmt.Sprintf("configuration-%d", i), "body": body, "received_at": when.Format(time.RFC3339Nano)}, 200)
		if profileString(t, a, "unique_id") != identity {
			t.Fatal("snapshot changed")
		}
		if i == 0 {
			firstID = profileString(t, a, "id")
		}
	}
	detail := h.call("GET", "/api/databases/"+databaseID, owner, nil, 200)
	if profileString(t, detail, "observed_unique_id") != "observed-new" || profileString(t, detail, "observed_unique_id_assessment_id") != firstID {
		t.Fatal("older/empty erased latest observation")
	}
	// Artificial excess relationships prove bounded history with a complete count.
	for i := 0; i < 7; i++ {
		if err := h.db.Create(&models.Ticket{ID: fmt.Sprintf("linked-%d", i), AssessmentID: firstID, ClientID: clientID, DatabaseID: databaseID, CheckType: "backups", Title: "Summary", Status: "open", CreatedAt: base}).Error; err != nil {
			t.Fatal(err)
		}
	}
	page := h.call("GET", "/api/databases/"+databaseID+"/assessments?limit=50", owner, nil, 200)
	h.call("GET", "/api/databases/"+databaseID+"/assessments?sort=created_at", owner, nil, 400)
	for _, forbidden := range []string{`"results"`, `"settings"`, `"raw_body"`, `"evidence"`} {
		if bytes.Contains(page["items"], []byte(forbidden)) {
			t.Fatal("history leaked", forbidden)
		}
	}
	var summaries []repository.AssessmentSummary
	if err := json.Unmarshal(page["items"], &summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 3 || summaries[0].TicketCount != 7 || len(summaries[0].Tickets) != 5 {
		t.Fatalf("bounded links %+v", summaries)
	}
	counter := &configurationQueryCounter{Interface: h.db.Logger}
	h.db.Logger = counter
	h.call("GET", "/api/databases/"+databaseID+"/assessments?limit=1", owner, nil, 200)
	oneRowQueries := counter.count
	counter.count = 0
	h.call("GET", "/api/databases/"+databaseID+"/assessments?limit=50", owner, nil, 200)
	if counter.count != oneRowQueries {
		t.Fatalf("history query count grew by rows: one=%d many=%d", oneRowQueries, counter.count)
	}
	links := h.call("GET", "/api/tickets?assessment_id="+firstID+"&limit=5&page=2", owner, nil, 200)
	if string(links["total"]) != "7" {
		t.Fatal("ticket relationship filtering")
	}
	_, normalLogin := h.createUser(owner, "view@example.test", "Viewer", "normal")
	normal := profileString(t, normalLogin, "access_token")
	h.call("PATCH", "/api/databases/"+databaseID, normal, map[string]any{"notes": "denied"}, 403)
	trustedID, trustedLogin := h.createUser(owner, "configure@example.test", "Configurer", "trusted")
	trusted := profileString(t, trustedLogin, "access_token")
	h.call("PATCH", "/api/databases/"+databaseID, trusted, map[string]any{"notes": "allowed"}, 200)
	h.call("DELETE", "/api/databases/"+databaseID, trusted, nil, 403)
	h.call("PATCH", "/api/admin/users/"+trustedID, owner, map[string]any{"status": "retired"}, 200)
	h.call("PATCH", "/api/databases/"+databaseID, trusted, map[string]any{"notes": "stale"}, 401)
}

func TestPostgresSourcePartialUpdatesAndFutureWindows(t *testing.T) {
	h := newProfileAPIHarness(t)
	owner := profileString(t, h.login("admin", "admin"), "access_token")
	client := h.call("POST", "/api/clients", owner, map[string]any{"name": "Scheduling"}, 201)
	d := h.call("POST", "/api/databases", owner, map[string]any{"client_id": profileString(t, client, "id"), "name": "SCHED_DB"}, 201)
	id := profileString(t, d, "id")
	h.call("PUT", "/api/databases/"+id+"/check-settings", owner, map[string]any{"selected": map[string]bool{"missing_email": true}}, 200)
	start := time.Date(2026, 10, 12, 5, 0, 0, 0, time.UTC) // Monday
	h.service.Now = func() time.Time { return start }
	source := h.call("POST", "/api/databases/"+id+"/email-sources", owner, map[string]any{"subject": "daily", "sender": "report@example.test", "mailbox": "inbox@example.test", "assessment_type": "daily_check", "timezone": "UTC", "enabled": true, "weekdays": []string{"mon"}, "schedules": []models.Schedule{{Name: "AM", Start: "06:00", End: "10:00"}}}, 200)
	sourceID := profileString(t, source, "id")
	h.service.Now = func() time.Time { return start.Add(time.Hour) }
	unchanged := h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"enabled": true}, 200)
	beforeTime, _ := time.Parse(time.RFC3339, profileString(t, source, "effective_at"))
	afterTime, _ := time.Parse(time.RFC3339, profileString(t, unchanged, "effective_at"))
	if !beforeTime.Equal(afterTime) {
		t.Fatalf("no-op moved effective date: before=%s after=%s", source["effective_at"], unchanged["effective_at"])
	}
	h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"assessment_type": "invented"}, 400)
	h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"weekdays": []string{"mon", "mon"}}, 400)
	h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"schedules": []models.Schedule{{Name: "AM", Start: "06:00", End: "11:00"}, {Name: "PM", Start: "10:00", End: "14:00"}}}, 400)
	h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"unexpected": true}, 400)
	page := h.call("GET", "/api/databases/"+id+"/email-sources?page=1&limit=50", owner, nil, 200)
	if string(page["total"]) != "1" {
		t.Fatal("source pagination")
	}
	h.service.Now = func() time.Time { return start.Add(4 * time.Hour) }
	early := h.call("POST", "/api/admin/schedules/evaluate", owner, map[string]any{"date": "2026-10-12"}, 200)
	if string(early["created"]) != "0" {
		t.Fatal("early alarm")
	}
	h.service.Now = func() time.Time { return start.Add(5 * time.Hour) }
	deadline := h.call("POST", "/api/admin/schedules/evaluate", owner, map[string]any{"date": "2026-10-12"}, 200)
	if string(deadline["created"]) != "1" {
		t.Fatal("absence not created")
	}
	again := h.call("POST", "/api/admin/schedules/evaluate", owner, map[string]any{"date": "2026-10-12"}, 200)
	if string(again["created"]) != "0" {
		t.Fatal("duplicate absence")
	}
	changed := h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"enabled": false, "mailbox": ""}, 200)
	if string(changed["enabled"]) != "false" || profileString(t, changed, "mailbox") != "" || profileString(t, changed, "subject") != "daily" {
		t.Fatal("partial fields not preserved/cleared")
	}
	h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"enabled": true, "schedules": []models.Schedule{{Name: "AM", Start: "06:00", End: "09:00"}}}, 200)
	retro := h.call("POST", "/api/admin/schedules/evaluate", owner, map[string]any{"date": "2026-10-12"}, 200)
	if string(retro["created"]) != "0" {
		t.Fatal("retroactive alarm after editing")
	}
	// Tuesday uses the configured start-day selection. Incomplete mail still
	// satisfies receipt while retaining its separate incomplete-report issue.
	h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"weekdays": []string{"tue"}, "schedules": []models.Schedule{{Name: "AM", Start: "06:00", End: "10:00"}}}, 200)
	h.call("POST", "/api/databases/"+id+"/assessments", owner, map[string]any{"message_id": "incomplete-in-window", "sender": "report@example.test", "subject": "daily", "body": "Script crashed", "received_at": "2026-10-13T06:00:00Z"}, 200)
	h.service.Now = func() time.Time { return start.Add(29 * time.Hour) }
	received := h.call("POST", "/api/admin/schedules/evaluate", owner, map[string]any{"date": "2026-10-13"}, 200)
	if string(received["created"]) != "0" {
		t.Fatal("incomplete receipt created second absence issue")
	}
	notSelected := h.call("POST", "/api/admin/schedules/evaluate", owner, map[string]any{"date": "2026-10-14"}, 200)
	if string(notSelected["created"]) != "0" {
		t.Fatal("unselected weekday evaluated")
	}
	// Adding inactive/standby expectations cannot produce new missing alarms.
	h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"assessment_type": "standby_check", "enabled": true, "weekdays": []string{"wed"}}, 200)
	h.service.Now = func() time.Time { return start.Add(53 * time.Hour) }
	standby := h.call("POST", "/api/admin/schedules/evaluate", owner, map[string]any{"date": "2026-10-14"}, 200)
	if string(standby["created"]) != "0" {
		t.Fatal("standby metadata evaluated")
	}
	h.call("PATCH", "/api/email-sources/"+sourceID, owner, map[string]any{"weekdays": []string{}, "assessment_type": "daily_check"}, 200)
	cleared := h.call("GET", "/api/email-sources/"+sourceID, owner, nil, 200)
	if string(cleared["weekdays"]) != "[]" {
		t.Fatal("empty days reverted to legacy all days")
	}
}
