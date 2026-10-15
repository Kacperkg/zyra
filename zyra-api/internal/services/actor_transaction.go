package services

import (
	"context"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

func requireParticipation(u models.User) error {
	if u.Status != models.StatusActive || !u.Role.Valid() {
		return apperrors.ErrForbidden
	}
	return nil
}

// lockActorWrite also supports operations which already own a transaction.
// Call it before locking domain rows; multi-user locks retain their own ordered path.
func (s *Service) lockActorWrite(ctx context.Context, tx repository.Store, actor models.User, permission func(models.User) error) (models.User, error) {
	var fresh models.User
	if err := tx.Users().Get(ctx, &fresh, actor.ID, true); err != nil {
		return fresh, err
	}
	if err := permission(fresh); err != nil {
		return fresh, err
	}
	return fresh, validateActorSession(ctx, tx, fresh.ID, s.Now())
}

// withActorWrite serializes participation with account retirement/role changes.
// Domain writes use the same transaction after checking the freshly locked user.
func (s *Service) withActorWrite(ctx context.Context, actor models.User, permission func(models.User) error, write func(*Service, models.User) error) error {
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		fresh, err := s.lockActorWrite(ctx, tx, actor, permission)
		if err != nil {
			return err
		}
		scoped := *s
		scoped.Store = tx
		return write(&scoped, fresh)
	})
}
