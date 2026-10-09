package repository

import (
	"context"
	"gorm.io/gorm"
	"zyra-api/internal/models"
)

type TicketEventRepository interface {
	Get(context.Context, *models.TicketEvent, string, bool) error
	Find(context.Context, *[]models.TicketEvent, Query) error
	Count(context.Context, Query) (int64, error)
	Create(context.Context, *models.TicketEvent) error
	Save(context.Context, *models.TicketEvent) error
	ParticipantUserIDs(context.Context, string) ([]string, error)
}
type ticketeventRepository struct{ entity[models.TicketEvent] }

func NewTicketEventRepository(db *gorm.DB) TicketEventRepository {
	return &ticketeventRepository{entity[models.TicketEvent]{db: db}}
}
func (s *store) TicketEvents() TicketEventRepository { return NewTicketEventRepository(s.db) }

func (r *ticketeventRepository) visible() *gorm.DB {
	return r.db.Where("(COALESCE(comment_id, '') = '' OR EXISTS (SELECT 1 FROM comments WHERE comments.id = ticket_events.comment_id AND comments.deleted_at IS NULL))")
}
func (r *ticketeventRepository) Find(ctx context.Context, out *[]models.TicketEvent, q Query) error {
	return (&entity[models.TicketEvent]{db: r.visible()}).Find(ctx, out, q)
}
func (r *ticketeventRepository) Count(ctx context.Context, q Query) (int64, error) {
	return (&entity[models.TicketEvent]{db: r.visible()}).Count(ctx, q)
}

func (r *ticketeventRepository) ParticipantUserIDs(ctx context.Context, ticketID string) ([]string, error) {
	ids := []string{}
	err := r.db.WithContext(ctx).Model(&models.TicketEvent{}).
		Distinct("user_id").Where("ticket_id = ? AND user_id <> ''", ticketID).
		Order("user_id").Pluck("user_id", &ids).Error
	return ids, translate(err)
}
