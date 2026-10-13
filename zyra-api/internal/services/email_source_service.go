package services

import (
	"context"
	"reflect"
	"strings"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type EmailSourceInput struct {
	Weekdays       *[]string          `json:"weekdays"`
	Sender         *string            `json:"sender"`
	Subject        *string            `json:"subject"`
	Mailbox        *string            `json:"mailbox"`
	AssessmentType *string            `json:"assessment_type"`
	Timezone       *string            `json:"timezone"`
	Enabled        *bool              `json:"enabled"`
	Schedules      *[]models.Schedule `json:"schedules"`
}

func (s *Service) SaveSourceInput(ctx context.Context, user models.User, databaseID, sourceID string, in EmailSourceInput) (models.EmailSource, error) {
	var result models.EmailSource
	err := s.withActorWrite(ctx, user, requireConfigure, func(scoped *Service, fresh models.User) error {
		value := models.EmailSource{AssessmentType: "daily_check", Weekdays: allWeekdays()}
		if sourceID != "" {
			if err := scoped.Store.EmailSources().Get(ctx, &value, sourceID, false); err != nil {
				return err
			}
			databaseID = value.DatabaseID
			var database models.Database
			if err := scoped.Store.Databases().Get(ctx, &database, databaseID, true); err != nil {
				return err
			}
			if err := scoped.Store.EmailSources().Get(ctx, &value, sourceID, true); err != nil {
				return err
			}
		}
		if in.Sender != nil {
			value.Sender = *in.Sender
		}
		if in.Subject != nil {
			value.Subject = *in.Subject
		}
		if in.Mailbox != nil {
			value.Mailbox = *in.Mailbox
		}
		if in.AssessmentType != nil {
			value.AssessmentType = *in.AssessmentType
		}
		if in.Timezone != nil {
			value.Timezone = *in.Timezone
		}
		if in.Enabled != nil {
			value.Enabled = *in.Enabled
		}
		if in.Schedules != nil {
			value.Schedules = *in.Schedules
		}
		if in.Weekdays != nil {
			value.Weekdays = *in.Weekdays
		}
		var err error
		result, err = scoped.saveSource(ctx, fresh, databaseID, value)
		return err
	})
	return result, err
}

func (s *Service) EmailSourcePage(ctx context.Context, databaseID string, page, limit int) (Page, error) {
	if _, err := s.Database(ctx, databaseID); err != nil {
		return Page{}, err
	}
	if page < 1 || page > 100000 || limit < 1 || limit > 200 {
		return Page{}, invalid("invalid pagination")
	}
	q := repository.Query{Where: map[string]any{"database_id": databaseID}, Order: "id asc", Limit: limit, Offset: (page - 1) * limit}
	return pageReader[models.EmailSource](s.Store.EmailSources())(ctx, q, ListOptions{Page: page, Limit: limit})
}

func (s *Service) EmailSource(ctx context.Context, id string) (models.EmailSource, error) {
	var source models.EmailSource
	err := s.Store.EmailSources().Get(ctx, &source, id, false)
	return source, err
}

func (s *Service) EmailSources(ctx context.Context, databaseID string) ([]models.EmailSource, error) {
	if _, err := s.Database(ctx, databaseID); err != nil {
		return nil, err
	}
	sources := []models.EmailSource{}
	count, err := s.Store.EmailSources().Count(ctx, repository.Query{Where: map[string]any{"database_id": databaseID}})
	if err != nil {
		return nil, err
	}
	if count > 200 {
		return nil, invalid("more than 200 sources; request page and limit for pagination")
	}
	err = s.Store.EmailSources().Find(ctx, &sources, repository.Query{Where: map[string]any{"database_id": databaseID}, Order: "id asc", Limit: 200})
	return sources, err
}

func (s *Service) DisableSource(ctx context.Context, user models.User, sourceID string) error {
	enabled := false
	_, err := s.SaveSourceInput(ctx, user, "", sourceID, EmailSourceInput{Enabled: &enabled})
	return err
}
func (s *Service) SourceSchedules(ctx context.Context, sourceID string) ([]models.Schedule, error) {
	source, err := s.EmailSource(ctx, sourceID)
	return source.Schedules, err
}

func (s *Service) SaveSourceSchedules(ctx context.Context, user models.User, sourceID string, schedules []models.Schedule) ([]models.Schedule, error) {
	result, err := s.SaveSourceInput(ctx, user, "", sourceID, EmailSourceInput{Schedules: &schedules})
	return result.Schedules, err
}
func (s *Service) saveSource(ctx context.Context, u models.User, dbID string, v models.EmailSource) (models.EmailSource, error) {
	if err := requireConfigure(u); err != nil {
		return v, err
	}
	var d models.Database
	if err := s.Store.Databases().Get(ctx, &d, dbID, true); err != nil {
		return v, err
	}
	if d.Archived {
		return v, apperrors.ErrConflict
	}
	var client models.Client
	if err := s.Store.Clients().Get(ctx, &client, d.ClientID, false); err != nil {
		return v, err
	}
	if client.Archived {
		return v, apperrors.ErrConflict
	}
	if v.Weekdays == nil {
		v.Weekdays = allWeekdays()
	}
	if err := validateWeekdays(v.Weekdays); err != nil {
		return v, err
	}
	sender, err := emailAddress(v.Sender)
	if err != nil {
		return v, err
	}
	v.Sender = sender
	if v.Mailbox != "" {
		v.Mailbox, err = emailAddress(v.Mailbox)
		if err != nil {
			return v, err
		}
	}
	if v.AssessmentType == "" {
		v.AssessmentType = "daily_check"
	}
	if v.AssessmentType != "daily_check" && v.AssessmentType != "standby_check" {
		return v, invalid("unsupported assessment_type")
	}
	v.Subject = strings.TrimSpace(v.Subject)
	if v.Subject == "" {
		return v, invalid("subject required")
	}
	if _, err = time.LoadLocation(v.Timezone); err != nil {
		return v, invalid("IANA timezone required")
	}
	seen := map[string]bool{}
	for _, w := range v.Schedules {
		if w.Name == "" || seen[w.Name] {
			return v, invalid("unique schedule name required")
		}
		seen[w.Name] = true
		if _, e := time.Parse("15:04", w.Start); e != nil {
			return v, invalid("start must be HH:MM")
		}
		if _, e := time.Parse("15:04", w.End); e != nil {
			return v, invalid("end must be HH:MM")
		}
		if w.Start == w.End {
			return v, invalid("window start and end must differ")
		}
	}
	if err := validateScheduleOverlap(v.Schedules); err != nil {
		return v, err
	}
	v.DatabaseID = dbID
	if v.ID != "" {
		var existing models.EmailSource
		if err = s.Store.EmailSources().Get(ctx, &existing, v.ID, false); err != nil {
			return v, err
		}
		if existing.DatabaseID != dbID {
			return v, apperrors.ErrConflict
		}
		v.EffectiveAt = existing.EffectiveAt
		if existing.Enabled != v.Enabled || existing.Timezone != v.Timezone || existing.Sender != v.Sender || existing.Subject != v.Subject || existing.Mailbox != v.Mailbox || existing.AssessmentType != v.AssessmentType || !reflect.DeepEqual(existing.Schedules, v.Schedules) || !reflect.DeepEqual(existing.Weekdays, v.Weekdays) {
			now := s.Now().UTC()
			v.EffectiveAt = &now
		}
		return v, s.Store.EmailSources().Save(ctx, &v)
	}
	v.ID = auth.Random()
	now := s.Now().UTC()
	v.EffectiveAt = &now
	return v, s.Store.EmailSources().Create(ctx, &v)
}
