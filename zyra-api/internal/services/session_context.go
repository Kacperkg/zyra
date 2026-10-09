package services

import (
	"context"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type sessionContextKey struct{}

// WithSessionContext carries the authenticated HTTP session into transactional
// service authorization without accepting a caller-controlled user/session ID.
func WithSessionContext(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, id)
}

// Call after locking the actor. Trusted internal jobs have no HTTP session.
func validateActorSession(ctx context.Context, tx repository.Store, userID string, now time.Time) error {
	id, exists := ctx.Value(sessionContextKey{}).(string)
	if !exists {
		return nil
	}
	if id == "" {
		return apperrors.ErrUnauthorized
	}
	var session models.Session
	if err := tx.Sessions().Get(ctx, &session, id, false); err != nil {
		return apperrors.ErrUnauthorized
	}
	if session.UserID != userID || session.Revoked || !now.Before(session.ExpiresAt) {
		return apperrors.ErrUnauthorized
	}
	return nil
}
