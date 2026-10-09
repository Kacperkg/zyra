package services

import (
	"context"
	"strings"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

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
	err := s.Store.EmailSources().Find(ctx, &sources, repository.Query{Where: map[string]any{"database_id": databaseID}, Order: "id asc", Limit: 200})
	return sources, err
}

func (s *Service) SaveSourceByID(ctx context.Context, user models.User, sourceID string, input models.EmailSource) (models.EmailSource, error) {
	var result models.EmailSource
	err := s.withActorWrite(ctx, user, requireConfigure, func(scoped *Service, fresh models.User) error {
		var err error
		result, err = scoped.saveSourceByID(ctx, fresh, sourceID, input)
		return err
	})
	return result, err
}
func (s *Service) saveSourceByID(ctx context.Context, user models.User, sourceID string, input models.EmailSource) (models.EmailSource, error) {
	existing, err := s.EmailSource(ctx, sourceID)
	if err != nil {
		return input, err
	}
	input.ID = existing.ID
	return s.saveSource(ctx, user, existing.DatabaseID, input)
}

func (s *Service) DisableSource(ctx context.Context, user models.User, sourceID string) error {
	return s.withActorWrite(ctx, user, requireConfigure, func(scoped *Service, fresh models.User) error { return scoped.disableSource(ctx, fresh, sourceID) })
}
func (s *Service) disableSource(ctx context.Context, user models.User, sourceID string) error {
	source, err := s.EmailSource(ctx, sourceID)
	if err != nil {
		return err
	}
	source.Enabled = false
	_, err = s.saveSource(ctx, user, source.DatabaseID, source)
	return err
}

func (s *Service) SourceSchedules(ctx context.Context, sourceID string) ([]models.Schedule, error) {
	source, err := s.EmailSource(ctx, sourceID)
	return source.Schedules, err
}

func (s *Service) SaveSourceSchedules(ctx context.Context, user models.User, sourceID string, schedules []models.Schedule) ([]models.Schedule, error) {
	var result []models.Schedule
	err := s.withActorWrite(ctx, user, requireConfigure, func(scoped *Service, fresh models.User) error {
		var err error
		result, err = scoped.saveSourceSchedules(ctx, fresh, sourceID, schedules)
		return err
	})
	return result, err
}
func (s *Service) saveSourceSchedules(ctx context.Context, user models.User, sourceID string, schedules []models.Schedule) ([]models.Schedule, error) {
	source, err := s.EmailSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	source.Schedules = schedules
	source, err = s.saveSource(ctx, user, source.DatabaseID, source)
	return source.Schedules, err
}

func (s *Service) SaveSource(ctx context.Context, u models.User, dbID string, v models.EmailSource) (models.EmailSource, error) {
	var result models.EmailSource
	err := s.withActorWrite(ctx, u, requireConfigure, func(scoped *Service, fresh models.User) error {
		var err error
		result, err = scoped.saveSource(ctx, fresh, dbID, v)
		return err
	})
	return result, err
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
	sender, err := emailAddress(v.Sender)
	if err != nil {
		return v, err
	}
	v.Sender = sender
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
	v.DatabaseID = dbID
	if v.ID != "" {
		var existing models.EmailSource
		if err = s.Store.EmailSources().Get(ctx, &existing, v.ID, false); err != nil {
			return v, err
		}
		if existing.DatabaseID != dbID {
			return v, apperrors.ErrConflict
		}
		return v, s.Store.EmailSources().Save(ctx, &v)
	}
	v.ID = auth.Random()
	return v, s.Store.EmailSources().Create(ctx, &v)
}
