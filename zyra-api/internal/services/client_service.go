package services

import (
	"context"
	"strings"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
)

type ClientInput struct {
	Name  *string `json:"name"`
	Notes *string `json:"notes"`
}

func (s *Service) SaveClient(ctx context.Context, u models.User, id string, in ClientInput) (models.Client, error) {
	var result models.Client
	err := s.withActorWrite(ctx, u, requireConfigure, func(scoped *Service, fresh models.User) error {
		var err error
		result, err = scoped.saveClient(ctx, fresh, id, in)
		return err
	})
	return result, err
}
func (s *Service) saveClient(ctx context.Context, u models.User, id string, in ClientInput) (models.Client, error) {
	var c models.Client
	if err := requireConfigure(u); err != nil {
		return c, err
	}
	if id != "" {
		if err := s.Store.Clients().Get(ctx, &c, id, true); err != nil {
			return c, err
		}
		if c.Archived {
			return c, apperrors.ErrConflict
		}
	} else {
		c.ID = auth.Random()
		c.CreatedAt = s.Now().UTC()
	}
	if in.Name != nil {
		c.Name = strings.TrimSpace(*in.Name)
	}
	if in.Notes != nil {
		c.Notes = *in.Notes
	}
	if c.Name == "" || len(c.Name) > 200 {
		return c, invalid("name must contain 1–200 bytes")
	}
	if id == "" {
		return c, s.Store.Clients().Create(ctx, &c)
	}
	return c, s.Store.Clients().Save(ctx, &c)
}
