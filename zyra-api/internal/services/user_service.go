package services

import (
	"context"
	"net/url"
	"strings"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type UserInput struct {
	Email      *string               `json:"email"`
	Name       *string               `json:"name"`
	Password   *string               `json:"password"`
	Role       *models.Role          `json:"role"`
	Status     *models.AccountStatus `json:"status"`
	Theme      *string               `json:"theme"`
	AvatarURL  *string               `json:"avatar_url"`
	Appearance *string               `json:"appearance"`
}

func (s *Service) SaveUser(ctx context.Context, actor models.User, id string, in UserInput) (models.User, error) {
	var u models.User
	err := s.Store.Transaction(ctx, func(tx repository.Store) error {
		ids := []string{actor.ID}
		if id != "" {
			ids = append(ids, id)
		}
		users, err := tx.Users().LockIDs(ctx, ids)
		if err != nil {
			return err
		}
		current, ok := users[actor.ID]
		if !ok || current.Status != models.StatusActive {
			return apperrors.ErrUnauthorized
		}
		if err := validateActorSession(ctx, tx, current.ID, s.Now()); err != nil {
			return err
		}
		self := id == actor.ID
		if !self {
			if err := requireAdmin(current); err != nil {
				return err
			}
		}
		if id != "" {
			var found bool
			u, found = users[id]
			if !found {
				return apperrors.ErrNotFound
			}
		} else {
			u = models.User{ID: auth.Random(), Role: models.RoleNormal, Status: models.StatusActive, Theme: "light", Appearance: "modern", CreatedAt: s.Now().UTC()}
		}
		prior := u
		// Check the original role so demoting and retiring together cannot bypass permission checks.
		if !self && current.Role != models.RoleOwner && u.Role.CanAdminister() {
			return apperrors.ErrForbidden
		}
		if in.Role != nil {
			if !in.Role.Valid() {
				return invalid("invalid role")
			}
			if *in.Role == models.RoleOwner || u.Role == models.RoleOwner || self {
				return apperrors.ErrForbidden
			}
			if current.Role != models.RoleOwner && (u.Role == models.RoleAdmin || *in.Role == models.RoleAdmin) {
				return apperrors.ErrForbidden
			}
			u.Role = *in.Role
		}
		if in.Status != nil {
			if *in.Status != models.StatusActive && *in.Status != models.StatusRetired {
				return invalid("invalid account status")
			}
			if self || u.Role == models.RoleOwner {
				return apperrors.ErrForbidden
			}
			if id == "" && *in.Status != models.StatusActive {
				return invalid("new accounts must be active")
			}
			u.Status = *in.Status
		}
		if in.Email != nil {
			u.Email, err = emailAddress(*in.Email)
			if err != nil {
				return err
			}
		}
		if in.Name != nil {
			u.Name = strings.TrimSpace(*in.Name)
			if len(u.Name) > 200 {
				return invalid("name exceeds 200 bytes")
			}
		}
		if in.Theme != nil {
			if *in.Theme != "light" && *in.Theme != "dark" {
				return invalid("invalid theme")
			}
			u.Theme = *in.Theme
		}
		if in.Appearance != nil {
			if *in.Appearance != "modern" && *in.Appearance != "classic" {
				return invalid("invalid appearance")
			}
			u.Appearance = *in.Appearance
		}
		if in.AvatarURL != nil {
			v := strings.TrimSpace(*in.AvatarURL)
			if v != "" {
				parsed, e := url.Parse(v)
				if e != nil || len(v) > 2048 || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Opaque != "" {
					return invalid("avatar_url must be an HTTPS URL without credentials")
				}
			}
			u.AvatarURL = v
		}
		if in.Password != nil {
			if id != "" {
				return invalid("use password recovery to change an existing password")
			}
			u.PasswordHash, err = passwordHash(*in.Password)
			if err != nil {
				return err
			}
		}
		if u.Email == "" || u.Name == "" || u.PasswordHash == "" {
			return invalid("email, name and password are required")
		}
		if id == "" {
			return tx.Users().Create(ctx, &u)
		}
		if err := tx.Users().Save(ctx, &u); err != nil {
			return err
		}
		if prior.Status != u.Status || prior.Role != u.Role {
			if err := tx.AccountLifecycle().Create(ctx, &models.AccountLifecycle{ID: auth.Random(), ActorID: current.ID, TargetID: u.ID, PreviousStatus: prior.Status, Status: u.Status, PreviousRole: prior.Role, Role: u.Role, CreatedAt: s.Now().UTC()}); err != nil {
				return err
			}
		}
		if u.Status == models.StatusRetired {
			if err := revokeSessions(ctx, tx, u.ID); err != nil {
				return err
			}
			tokens := []models.ResetToken{}
			if err := tx.ResetTokens().Find(ctx, &tokens, repository.Query{Where: map[string]any{"user_id": u.ID}}); err != nil {
				return err
			}
			for _, token := range tokens {
				token.Used = true
				if err := tx.ResetTokens().Save(ctx, &token); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return u, err
}
