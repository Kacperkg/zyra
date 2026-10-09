package services

import (
	"context"
	"fmt"
	"strings"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type ReportInput struct {
	MessageID  string    `json:"message_id"`
	Subject    string    `json:"subject"`
	Sender     string    `json:"sender"`
	Body       string    `json:"body"`
	ReceivedAt time.Time `json:"received_at"`
}

func (s *Service) AssessmentRawBody(ctx context.Context, id string) (string, error) {
	assessment, err := s.Assessment(ctx, id)
	return assessment.RawBody, err
}

func (s *Service) SubmitReport(ctx context.Context, u models.User, dbID string, in ReportInput) (models.Assessment, error) {
	var a models.Assessment
	if err := requireAdmin(u); err != nil {
		return a, err
	}
	if strings.TrimSpace(in.MessageID) == "" || strings.TrimSpace(in.Body) == "" {
		return a, invalid("message_id and body required")
	}
	if in.ReceivedAt.IsZero() {
		in.ReceivedAt = s.Now().UTC()
	}
	var d models.Database
	if err := s.Store.Databases().Get(ctx, &d, dbID, false); err != nil {
		return a, err
	}
	if d.Archived {
		return a, apperrors.ErrConflict
	}
	if m := databaseNameRE.FindStringSubmatch(in.Body); len(m) > 1 && !strings.EqualFold(strings.TrimSpace(m[1]), d.Name) {
		return a, invalid("report database does not match the selected database")
	}
	if in.Sender != "" {
		e, err := emailAddress(in.Sender)
		if err != nil {
			return a, err
		}
		in.Sender = e
	}
	in.Subject = strings.TrimSpace(in.Subject)
	settings, err := s.GetSettings(ctx, dbID)
	if err != nil {
		return a, err
	}
	parsed := ParseReport(in.Body, settings)
	a = models.Assessment{ID: auth.Random(), DatabaseID: dbID, MessageID: in.MessageID, Type: "daily_check", Subject: in.Subject, Sender: in.Sender, ReceivedAt: in.ReceivedAt.UTC(), ReportTime: parsed.ReportTime, Hostname: parsed.Hostname, RawBody: in.Body, Settings: settings, Results: parsed.Results, Result: "passed"}
	if parsed.Hostname != "" && strings.EqualFold(parsed.Hostname, d.Hostname) {
		a.IP = d.IP
	}
	evaluated := false
	for _, r := range a.Results {
		if r.Status == "failed" {
			a.Result = "failed"
		}
		if r.Status == "passed" || r.Status == "failed" {
			evaluated = true
		}
		if r.Status == "unknown" && a.Result != "failed" {
			a.Result = "unknown"
		}
	}
	if !parsed.Complete {
		a.Result = "incomplete"
	} else if !evaluated && a.Result == "passed" {
		a.Result = "not_evaluated"
	}
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
		if e := tx.Databases().Get(ctx, &locked, dbID, true); e != nil {
			return e
		}
		if locked.Archived {
			return apperrors.ErrConflict
		}
		existing := []models.Assessment{}
		if e := tx.Assessments().Find(ctx, &existing, repository.Query{Where: map[string]any{"message_id": in.MessageID}, Limit: 1}); e != nil {
			return e
		}
		if len(existing) > 0 {
			if existing[0].DatabaseID != dbID || existing[0].RawBody != in.Body {
				return apperrors.ErrConflict
			}
			a = existing[0]
			return nil
		}
		if e := tx.Assessments().Create(ctx, &a); e != nil {
			return e
		}
		for _, findings := range groupedFailures(a.Results) {
			first := findings[0]
			resources := []string{}
			evidence := []string{}
			for _, finding := range findings {
				if finding.Resource != "" {
					resources = append(resources, finding.Resource)
				}
				if strings.TrimSpace(finding.Evidence) != "" {
					evidence = append(evidence, finding.Evidence)
				}
			}
			now := s.Now().UTC()
			t := models.Ticket{ID: auth.Random(), ClientID: d.ClientID, DatabaseID: dbID, AssessmentID: a.ID, AssessmentType: "daily_check", CheckType: first.CheckType, Resource: strings.Join(resources, ", "), Title: fmt.Sprintf("%s: %s — %s", displayCheckType(first.CheckType), d.Name, first.Summary), Status: "open", Hostname: a.Hostname, IP: a.IP, Evidence: strings.Join(evidence, "\n"), CreatedAt: now, UpdatedAt: now}
			if e := tx.Tickets().Create(ctx, &t); e != nil {
				return e
			}
			if e := tx.TicketEvents().Create(ctx, &models.TicketEvent{ID: auth.Random(), TicketID: t.ID, Type: "system_findings", Comment: findingsComment(findings), CreatedAt: t.CreatedAt}); e != nil {
				return e
			}
		}
		return nil
	})
	return a, err
}
