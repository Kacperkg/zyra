package services

import (
	"context"
	"fmt"
	"time"
	"zyra-api/internal/apperrors"
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
		loc, err := time.LoadLocation(source.Timezone)
		if err != nil {
			return created, err
		}
		day, err := time.ParseInLocation("2006-01-02", date, loc)
		if err != nil {
			return created, invalid("date must be YYYY-MM-DD")
		}
		var d models.Database
		if err = s.Store.Databases().Get(ctx, &d, source.DatabaseID, false); err != nil {
			return created, err
		}
		if d.Archived {
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
				existing := []models.Ticket{}
				if e := tx.Tickets().Find(ctx, &existing, repository.Query{Where: map[string]any{"id": id}, Limit: 1}); e != nil {
					return e
				}
				if len(existing) > 0 {
					return nil
				}
				reports := []models.Assessment{}
				if e := tx.Assessments().Find(ctx, &reports, repository.Query{Where: map[string]any{"database_id": d.ID, "sender": source.Sender, "subject": source.Subject}, TimeField: "received_at", After: &start, Before: &end}); e != nil {
					return e
				}
				// A received incomplete report already represents this failed run.
				if len(reports) > 0 {
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
	start, err := time.ParseInLocation("2006-01-02 15:04", day.Format("2006-01-02")+" "+w.Start, day.Location())
	if err != nil {
		return start, start, apperrors.ErrInvalidInput
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", day.Format("2006-01-02")+" "+w.End, day.Location())
	if err != nil {
		return start, end, apperrors.ErrInvalidInput
	}
	if !end.After(start) {
		end = end.AddDate(0, 0, 1)
	}
	return start, end, nil
}
