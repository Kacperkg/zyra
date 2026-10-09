package repository

import (
	"context"
	"gorm.io/gorm"
)

// Store groups typed repositories and keeps cross-domain writes in one transaction.
type Store interface {
	Users() UserRepository
	Clients() ClientRepository
	Databases() DatabaseRepository
	Settings() DatabaseSettingsRepository
	Sessions() SessionRepository
	ResetTokens() ResetTokenRepository
	EmailSources() EmailSourceRepository
	Assessments() AssessmentRepository
	Tickets() TicketRepository
	TicketEvents() TicketEventRepository
	Comments() CommentRepository
	SavedTickets() SavedTicketRepository
	Mentions() MentionRepository
	Notifications() NotificationRepository
	AccountLifecycle() AccountLifecycleRepository
	Transaction(context.Context, func(Store) error) error
}
type store struct{ db *gorm.DB }

func New(db *gorm.DB) Store { return &store{db: db} }
func (s *store) Transaction(ctx context.Context, fn func(Store) error) error {
	return translate(s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(&store{db: tx}) }))
}
