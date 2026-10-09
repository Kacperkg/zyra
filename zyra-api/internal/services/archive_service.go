package services

import (
	"context"
	"fmt"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

func (s *Service) Archive(ctx context.Context, u models.User, kind, id string) error {
	return s.withActorWrite(ctx, u, requireAdmin, func(scoped *Service, fresh models.User) error { return scoped.archive(ctx, fresh, kind, id) })
}
func (s *Service) archive(ctx context.Context, u models.User, kind, id string) error {
	if err := requireAdmin(u); err != nil {
		return err
	}
	if kind == "clients" {
		var c models.Client
		if err := s.Store.Clients().Get(ctx, &c, id, true); err != nil {
			return err
		}
		n, err := s.Store.Databases().Count(ctx, repository.Query{Where: map[string]any{"client_id": id, "archived": false}})
		if err != nil {
			return err
		}
		if n > 0 {
			return fmt.Errorf("%w: archive client databases first", apperrors.ErrConflict)
		}
		c.Archived = true
		return s.Store.Clients().Save(ctx, &c)
	}
	var d models.Database
	if err := s.Store.Databases().Get(ctx, &d, id, true); err != nil {
		return err
	}
	d.Archived = true
	return s.Store.Databases().Save(ctx, &d)
}
