package services

import (
	"context"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

// withActorWrite serializes participation with account retirement/role changes.
// Domain writes use the same transaction after checking the freshly locked user.
func (s *Service) withActorWrite(ctx context.Context, actor models.User, permission func(models.User) error, write func(*Service, models.User) error) error {
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		var fresh models.User
		if err := tx.Users().Get(ctx, &fresh, actor.ID, true); err != nil {
			return err
		}
		if err := permission(fresh); err != nil {
			return err
		}
		if err := validateActorSession(ctx, tx, fresh.ID, s.Now()); err != nil {
			return err
		}
		scoped := *s
		scoped.Store = tx
		return write(&scoped, fresh)
	})
}
