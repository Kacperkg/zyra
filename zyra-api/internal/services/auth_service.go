package services

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type TokenPair struct {
	AccessToken      string      `json:"access_token"`
	RefreshToken     string      `json:"refresh_token"`
	AccessExpiresAt  time.Time   `json:"access_expires_at"`
	SessionExpiresAt time.Time   `json:"session_expires_at"`
	User             models.User `json:"user"`
}

func (s *Service) Bootstrap(ctx context.Context, email, password string) error {
	e, err := emailAddress(email)
	if err != nil {
		return err
	}
	users := []models.User{}
	if err = s.Store.Users().Find(ctx, &users, repository.Query{Where: map[string]any{"email": e}, Limit: 1}); err != nil {
		return err
	}
	if len(users) > 0 {
		return nil
	}
	hash, err := passwordHash(password)
	if err != nil {
		return err
	}
	return s.Store.Users().Create(ctx, &models.User{ID: auth.Random(), Email: e, Name: "Administrator", Role: models.RoleAdmin, Theme: "light", PasswordHash: hash, CreatedAt: s.Now().UTC()})
}
func (s *Service) Login(ctx context.Context, email, password string) (TokenPair, error) {
	var result TokenPair
	if s.DevAdminLogin && strings.EqualFold(strings.TrimSpace(email), "admin") {
		email = developmentAdminEmail
	}
	e, err := emailAddress(email)
	if err != nil {
		return result, apperrors.ErrUnauthorized
	}
	users := []models.User{}
	if err = s.Store.Users().Find(ctx, &users, repository.Query{Where: map[string]any{"email": e}, Limit: 1}); err != nil {
		return result, err
	}
	if len(users) != 1 || users[0].Disabled {
		return result, apperrors.ErrUnauthorized
	}
	u := users[0]
	if u.ID == developmentAdminID && !s.DevAdminLogin {
		return result, apperrors.ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return result, apperrors.ErrUnauthorized
	}
	raw := auth.Random()
	session := models.Session{ID: auth.Random(), UserID: u.ID, TokenHash: auth.Hash(raw), ExpiresAt: s.Now().UTC().Truncate(time.Microsecond).Add(7 * 24 * time.Hour)}
	token, expires, err := s.Tokens.Issue(u.ID, session.ID, session.ExpiresAt)
	if err != nil {
		return result, err
	}
	if err = s.Store.Sessions().Create(ctx, &session); err != nil {
		return result, err
	}
	return TokenPair{token, raw, expires, session.ExpiresAt, u}, nil
}
func (s *Service) Refresh(ctx context.Context, raw string) (TokenPair, error) {
	var pair TokenPair
	if raw == "" {
		return pair, apperrors.ErrUnauthorized
	}
	err := s.Store.Transaction(ctx, func(tx repository.Store) error {
		sessions := []models.Session{}
		if err := tx.Sessions().Find(ctx, &sessions, repository.Query{Where: map[string]any{"token_hash": auth.Hash(raw)}, Limit: 1}); err != nil {
			return err
		}
		if len(sessions) != 1 {
			return apperrors.ErrUnauthorized
		}
		var session models.Session
		if err := tx.Sessions().Get(ctx, &session, sessions[0].ID, true); err != nil {
			return err
		}
		if session.Revoked || session.TokenHash != auth.Hash(raw) || !s.Now().Before(session.ExpiresAt) {
			return apperrors.ErrUnauthorized
		}
		session.ExpiresAt = session.ExpiresAt.UTC()
		var u models.User
		if err := tx.Users().Get(ctx, &u, session.UserID, false); err != nil {
			return err
		}
		if u.Disabled || (u.ID == developmentAdminID && !s.DevAdminLogin) {
			return apperrors.ErrUnauthorized
		}
		token, expires, err := s.Tokens.Issue(u.ID, session.ID, session.ExpiresAt)
		if err != nil {
			return err
		}
		next := auth.Random()
		session.TokenHash = auth.Hash(next)
		if err = tx.Sessions().Save(ctx, &session); err != nil {
			return err
		}
		pair = TokenPair{token, next, expires, session.ExpiresAt, u}
		return nil
	})
	return pair, err
}
func (s *Service) Authenticate(ctx context.Context, raw string) (models.User, string, error) {
	var u models.User
	claims, err := s.Tokens.Parse(raw)
	if err != nil {
		return u, "", apperrors.ErrUnauthorized
	}
	var session models.Session
	if err = s.Store.Sessions().Get(ctx, &session, claims.SessionID, false); err != nil {
		return u, "", apperrors.ErrUnauthorized
	}
	if session.Revoked || session.UserID != claims.Subject || !s.Now().Before(session.ExpiresAt) {
		return u, "", apperrors.ErrUnauthorized
	}
	if err = s.Store.Users().Get(ctx, &u, claims.Subject, false); err != nil || u.Disabled || (u.ID == developmentAdminID && !s.DevAdminLogin) {
		return u, "", apperrors.ErrUnauthorized
	}
	return u, session.ID, nil
}
func (s *Service) Logout(ctx context.Context, id string) error {
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		var session models.Session
		if err := tx.Sessions().Get(ctx, &session, id, true); err != nil {
			return err
		}
		session.Revoked = true
		return tx.Sessions().Save(ctx, &session)
	})
}
func revokeSessions(ctx context.Context, tx repository.Store, userID string) error {
	sessions := []models.Session{}
	if err := tx.Sessions().Find(ctx, &sessions, repository.Query{Where: map[string]any{"user_id": userID}}); err != nil {
		return err
	}
	for _, session := range sessions {
		session.Revoked = true
		if err := tx.Sessions().Save(ctx, &session); err != nil {
			return err
		}
	}
	return nil
}

var ErrRecoveryUnavailable = errors.New("password recovery delivery is not configured")

func (s *Service) ChangePassword(ctx context.Context, u models.User, current, password string) error {
	hash, err := passwordHash(password)
	if err != nil {
		return err
	}
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		var fresh models.User
		if err := tx.Users().Get(ctx, &fresh, u.ID, true); err != nil {
			return err
		}
		if bcrypt.CompareHashAndPassword([]byte(fresh.PasswordHash), []byte(current)) != nil {
			return apperrors.ErrUnauthorized
		}
		fresh.PasswordHash = hash
		if err := tx.Users().Save(ctx, &fresh); err != nil {
			return err
		}
		return revokeSessions(ctx, tx, u.ID)
	})
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	if s.Recovery == nil {
		return ErrRecoveryUnavailable
	}
	e, err := emailAddress(email)
	if err != nil {
		return nil
	}
	users := []models.User{}
	if err = s.Store.Users().Find(ctx, &users, repository.Query{Where: map[string]any{"email": e}, Limit: 1}); err != nil {
		return err
	}
	if len(users) != 1 || users[0].Disabled {
		return nil
	}
	raw := auth.Random()
	reset := models.ResetToken{ID: auth.Random(), UserID: users[0].ID, TokenHash: auth.Hash(raw), ExpiresAt: s.Now().Add(30 * time.Minute)}
	if err = s.Store.ResetTokens().Create(ctx, &reset); err != nil {
		return err
	}
	return s.Recovery.SendReset(ctx, e, raw)
}
func (s *Service) ResetPassword(ctx context.Context, raw, password string) error {
	hash, err := passwordHash(password)
	if err != nil {
		return err
	}
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		tokens := []models.ResetToken{}
		if err := tx.ResetTokens().Find(ctx, &tokens, repository.Query{Where: map[string]any{"token_hash": auth.Hash(raw)}, Limit: 1}); err != nil {
			return err
		}
		if len(tokens) != 1 {
			return apperrors.ErrUnauthorized
		}
		var token models.ResetToken
		if err := tx.ResetTokens().Get(ctx, &token, tokens[0].ID, true); err != nil {
			return err
		}
		if token.Used || !s.Now().Before(token.ExpiresAt) {
			return apperrors.ErrUnauthorized
		}
		var user models.User
		if err := tx.Users().Get(ctx, &user, token.UserID, true); err != nil {
			return err
		}
		if user.Disabled {
			return apperrors.ErrUnauthorized
		}
		user.PasswordHash = hash
		token.Used = true
		if err := tx.Users().Save(ctx, &user); err != nil {
			return err
		}
		if err := tx.ResetTokens().Save(ctx, &token); err != nil {
			return err
		}
		return revokeSessions(ctx, tx, user.ID)
	})
}
