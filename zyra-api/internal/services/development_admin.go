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
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.Store.Users().Create(ctx, &models.User{ID: developmentAdminID, Email: developmentAdminEmail, Name: "Development Administrator", Role: models.RoleAdmin, Theme: "light", PasswordHash: string(hash), CreatedAt: s.Now().UTC()})
}
