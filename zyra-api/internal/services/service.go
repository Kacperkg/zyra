package services

import (
	"context"
	"time"
	"zyra-api/internal/auth"
	"zyra-api/internal/repository"
)

type RecoverySender interface {
	SendReset(context.Context, string, string) error
}
type Service struct {
	Store         repository.Store
	Tokens        *auth.Tokens
	Now           func() time.Time
	Recovery      RecoverySender
	DevAdminLogin bool
}

func New(store repository.Store, tokens *auth.Tokens) *Service {
	return &Service{Store: store, Tokens: tokens, Now: time.Now}
}
