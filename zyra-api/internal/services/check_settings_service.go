package services

import (
	"context"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

func (s *Service) GetSettings(ctx context.Context, id string) (models.CheckSettings, error) {
	var d models.Database
	if err := s.Store.Databases().Get(ctx, &d, id, false); err != nil {
		return models.CheckSettings{}, err
	}
	rows := []models.DatabaseSettings{}
	if err := s.Store.Settings().Find(ctx, &rows, repository.Query{Where: map[string]any{"id": id}, Limit: 1}); err != nil {
		return models.CheckSettings{}, err
	}
	if len(rows) == 0 {
		return models.CheckSettings{Selected: map[string]bool{}, Resources: map[string]map[string]models.ResourceRule{}}, nil
	}
	return rows[0].Settings, nil
}
func (s *Service) PutSettings(ctx context.Context, u models.User, id string, v models.CheckSettings) error {
	return s.withActorWrite(ctx, u, requireConfigure, func(scoped *Service, fresh models.User) error { return scoped.putSettings(ctx, fresh, id, v) })
}
func (s *Service) putSettings(ctx context.Context, u models.User, id string, v models.CheckSettings) error {
	if err := requireConfigure(u); err != nil {
		return err
	}
	var d models.Database
	if err := s.Store.Databases().Get(ctx, &d, id, true); err != nil {
		return err
	}
	if d.Archived {
		return apperrors.ErrConflict
	}
	if err := ValidateSettings(v); err != nil {
		return err
	}
	return s.Store.Settings().Save(ctx, &models.DatabaseSettings{ID: id, Settings: v})
}
