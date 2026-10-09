package services

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

const developmentAdminID = "zyra-development-admin"
const developmentAdminEmail = "admin@zyra.test"

// BootstrapDevelopmentAdmin is an explicit local test fixture. It never resets
// an existing account or weakens validation for ordinary account creation.
func (s *Service) BootstrapDevelopmentAdmin(ctx context.Context) error {
	if !s.DevAdminLogin {
		return apperrors.ErrForbidden
	}
	users := []models.User{}
	if err := s.Store.Users().Find(ctx, &users, repository.Query{Where: map[string]any{"email": developmentAdminEmail}, Limit: 1}); err != nil {
		return err
	}
	if len(users) > 0 {
		if users[0].ID != developmentAdminID {
			return apperrors.ErrConflict
		}
		return s.EnsureOwner(ctx, developmentAdminEmail)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	owners, err := s.Store.Users().Count(ctx, repository.Query{Where: map[string]any{"role": models.RoleOwner}})
	if err != nil {
		return err
	}
	if owners != 0 {
		return apperrors.ErrConflict
	}
	return s.Store.Users().Create(ctx, &models.User{ID: developmentAdminID, Email: developmentAdminEmail, Name: "Development Administrator", Role: models.RoleOwner, Status: models.StatusActive, Appearance: "modern", Theme: "light", PasswordHash: string(hash), CreatedAt: s.Now().UTC()})
}

// EnsureOwner requires explicit selection when no owner exists. It never picks
// an arbitrary existing administrator and never demotes/replaces an owner.
func (s *Service) EnsureOwner(ctx context.Context, email string) error {
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		owners := []models.User{}
		if err := tx.Users().Find(ctx, &owners, repository.Query{Where: map[string]any{"role": models.RoleOwner}, Limit: 2}); err != nil {
			return err
		}
		if len(owners) > 1 {
			return apperrors.ErrConflict
		}
		if len(owners) == 1 {
			if owners[0].Status != models.StatusActive {
				return apperrors.ErrConflict
			}
			if email != "" {
				canonical, err := emailAddress(email)
				if err != nil {
					return err
				}
				if owners[0].Email != canonical {
					return invalid("OWNER_EMAIL does not match the existing owner")
				}
			}
			return nil
		}
		if email == "" {
			return invalid("OWNER_EMAIL must explicitly select an existing active account before startup")
		}
		canonical, err := emailAddress(email)
		if err != nil {
			return err
		}
		users := []models.User{}
		if err := tx.Users().Find(ctx, &users, repository.Query{Where: map[string]any{"email": canonical}, Limit: 1}); err != nil {
			return err
		}
		if len(users) != 1 {
			return invalid("OWNER_EMAIL must identify an existing account")
		}
		var target models.User
		if err := tx.Users().Get(ctx, &target, users[0].ID, true); err != nil {
			return err
		}
		if target.Status != models.StatusActive {
			return invalid("owner must be active")
		}
		target.Role = models.RoleOwner
		return tx.Users().Save(ctx, &target)
	})
}
