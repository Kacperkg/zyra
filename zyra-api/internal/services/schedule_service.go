package services

import (
	"context"
	"fmt"
	"reflect"
	"time"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

func (s *Service) EvaluateSchedules(ctx context.Context, u models.User, date string) (int, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return 0, invalid("date must be YYYY-MM-DD")
	}
	if err := requireAdmin(u); err != nil {
		return 0, err
	}
	sources := []models.EmailSource{}
	if err := s.Store.EmailSources().Find(ctx, &sources, repository.Query{Where: map[string]any{"enabled": true}}); err != nil {
		return 0, err
	}
	created := 0
	for _, source := range sources {
		if source.AssessmentType != "" && source.AssessmentType != "daily_check" {
			continue
		}
		loc, err := time.LoadLocation(source.Timezone)
		if err != nil {
			return created, err
		}
		day, err := time.ParseInLocation("2006-01-02", date, loc)
		if err != nil {
			return created, invalid("date must be YYYY-MM-DD")
		}
		if !scheduledDay(source.Weekdays, day.Weekday()) {
			continue
		}
		var d models.Database
		if err = s.Store.Databases().Get(ctx, &d, source.DatabaseID, false); err != nil {
			return created, err
		}
		if d.Archived || (d.Engine != "" && d.Engine != "oracle") {
			continue
		}
		var client models.Client
		if err = s.Store.Clients().Get(ctx, &client, d.ClientID, false); err != nil {
			return created, err
		}
		if client.Archived {
			continue
		}
		settings, err := s.GetSettings(ctx, d.ID)
		if err != nil {
			return created, err
		}
		if !settings.Selected["missing_email"] {
			continue
		}
		for _, w := range source.Schedules {
			start, end, err := scheduleBounds(day, w)
			if err != nil {
				return created, err
			}
			if s.Now().Before(end) {
				continue
			}
			if source.EffectiveAt != nil && start.Before(*source.EffectiveAt) {
				continue
			}
			id := "missing:" + source.ID + ":" + start.UTC().Format(time.RFC3339) + ":" + end.UTC().Format(time.RFC3339)
			made := false
			err = s.Store.Transaction(ctx, func(tx repository.Store) error {
				var actor models.User
				if e := tx.Users().Get(ctx, &actor, u.ID, true); e != nil {
					return e
				}
				if e := requireAdmin(actor); e != nil {
					return e
				}
				if e := validateActorSession(ctx, tx, actor.ID, s.Now()); e != nil {
					return e
				}
				var locked models.Database
				if e := tx.Databases().Get(ctx, &locked, d.ID, true); e != nil {
					return e
				}
				if locked.Archived {
					return nil
				}
				var currentClient models.Client
				if e := tx.Clients().Get(ctx, &currentClient, d.ClientID, false); e != nil {
					return e
				}
				if currentClient.Archived {
					return nil
				}
				currentSettings, e := (&Service{Store: tx}).GetSettings(ctx, d.ID)
				if e != nil {
					return e
				}
				if !currentSettings.Selected["missing_email"] {
					return nil
				}
				var current models.EmailSource
				if e := tx.EmailSources().Get(ctx, &current, source.ID, true); e != nil {
					return e
				}
				if !reflect.DeepEqual(current, source) {
					return nil
				}
				existing := []models.Ticket{}
				if e := tx.Tickets().Find(ctx, &existing, repository.Query{Where: map[string]any{"id": id}, Limit: 1}); e != nil {
					return e
				}
				if len(existing) > 0 {
					return nil
				}
				received, e := tx.Assessments().Exists(ctx, repository.Query{Where: map[string]any{"database_id": d.ID, "sender": source.Sender, "subject": source.Subject}, TimeField: "received_at", After: &start, Before: &end})
				if e != nil {
					return e
				}
				// A received incomplete report already represents this failed run.
				if received {
					return nil
				}
				t := models.Ticket{ID: id, ClientID: d.ClientID, DatabaseID: d.ID, AssessmentType: "missing", CheckType: "missing_email", Title: fmt.Sprintf("Missing email: %s — %s", d.Name, w.Name), Status: "open", Hostname: d.Hostname, IP: d.IP, Evidence: fmt.Sprintf("No matching report received between %s and %s (%s). Server details are configured, not observed.", start.Format(time.RFC3339), end.Format(time.RFC3339), source.Timezone), CreatedAt: s.Now().UTC(), UpdatedAt: s.Now().UTC()}
				if e := tx.Tickets().Create(ctx, &t); e != nil {
					return e
				}
				made = true
				finding := models.CheckResult{CheckType: "missing_email", Status: "failed", Summary: "No email received in the expected window", Evidence: t.Evidence}
				return tx.TicketEvents().Create(ctx, &models.TicketEvent{ID: id + ":findings", TicketID: id, Type: "system_findings", Comment: findingsComment([]models.CheckResult{finding}), CreatedAt: t.CreatedAt})
			})
			if err != nil {
				return created, err
			}
			if made {
				created++
			}
		}
	}
	return created, nil
}
func scheduleBounds(day time.Time, w models.Schedule) (time.Time, time.Time, error) {
	start, err := strictLocalTime(day, w.Start)
	if err != nil {
		return start, start, err
	}
	endDay := day
	if w.End <= w.Start {
		endDay = day.AddDate(0, 0, 1)
	}
	end, err := strictLocalTime(endDay, w.End)
	if err != nil {
		return start, end, err
	}
	return start, end, nil
}

func strictLocalTime(day time.Time, clock string) (time.Time, error) {
	want := day.Format("2006-01-02") + " " + clock
	value, err := time.ParseInLocation("2006-01-02 15:04", want, day.Location())
	if err != nil || value.Format("2006-01-02 15:04") != want {
		return time.Time{}, invalid("schedule boundary is nonexistent in source timezone")
	}
	// An alternate UTC offset near this date may map the same wall time twice.
	_, offset := value.Zone()
	for _, delta := range []time.Duration{-48 * time.Hour, -24 * time.Hour, 24 * time.Hour, 48 * time.Hour} {
		_, alternate := value.Add(delta).Zone()
		candidate := value.Add(time.Duration(offset-alternate) * time.Second)
		if alternate != offset && candidate.Format("2006-01-02 15:04") == want {
			return time.Time{}, invalid("schedule boundary is ambiguous in source timezone")
		}
	}
	return value, nil
}
func allWeekdays() []string { return []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"} }
func validateWeekdays(days []string) error {
	valid := map[string]bool{}
	for _, day := range allWeekdays() {
		valid[day] = true
	}
	seen := map[string]bool{}
	for _, day := range days {
		if !valid[day] || seen[day] {
			return invalid("weekdays must contain unique mon..sun values")
		}
		seen[day] = true
	}
	return nil
}
func scheduledDay(days []string, day time.Weekday) bool {
	if days == nil {
		return true
	} // preserved legacy all-day semantics
	want := []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}[day]
	for _, value := range days {
		if value == want {
			return true
		}
	}
	return false
}
func validateScheduleOverlap(windows []models.Schedule) error {
	occupied := make([]bool, 1440)
	for _, window := range windows {
		start, err := time.Parse("15:04", window.Start)
		if err != nil {
			return invalid("start must be HH:MM")
		}
		end, err := time.Parse("15:04", window.End)
		if err != nil {
			return invalid("end must be HH:MM")
		}
		from, to := start.Hour()*60+start.Minute(), end.Hour()*60+end.Minute()
		if from == to {
			return invalid("window start and end must differ")
		}
		for minute := from; minute != to; minute = (minute + 1) % 1440 {
			if occupied[minute] {
				return invalid("schedule windows must not overlap")
			}
			occupied[minute] = true
		}
	}
	return nil
}
