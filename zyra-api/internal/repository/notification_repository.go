package repository

import (
	"context"
	"gorm.io/gorm"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
)

type NotificationRow struct {
	ID             string     `json:"id"`
	TicketID       string     `json:"ticket_id"`
	TicketNumber   int64      `json:"ticket_number"`
	TicketTitle    string     `json:"ticket_title"`
	CommentID      string     `json:"comment_id"`
	ActorID        string     `json:"actor_id"`
	ActorName      string     `json:"actor_name"`
	ActorAvatarURL string     `json:"actor_avatar_url"`
	CreatedAt      time.Time  `json:"created_at"`
	ReadAt         *time.Time `json:"read_at"`
}
type NotificationRepository interface {
	List(context.Context, string, bool, int, int) ([]NotificationRow, int64, error)
	UnreadCount(context.Context, string) (int64, error)
	Read(context.Context, string, string, time.Time) error
	ReadAll(context.Context, string, time.Time) error
}
type notificationRepository struct{ db *gorm.DB }

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db}
}
func (s *store) Notifications() NotificationRepository { return NewNotificationRepository(s.db) }
func (r *notificationRepository) List(ctx context.Context, userID string, unread bool, limit, offset int) ([]NotificationRow, int64, error) {
	rows := []NotificationRow{}
	db := r.db.WithContext(ctx).Table("notifications").Joins("JOIN tickets ON tickets.id = notifications.ticket_id").Joins("JOIN users actor ON actor.id = notifications.actor_id").Where("notifications.recipient_id = ? AND notifications.withdrawn_at IS NULL", userID)
	if unread {
		db = db.Where("notifications.read_at IS NULL")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return rows, 0, translate(err)
	}
	err := db.Select("notifications.id, notifications.ticket_id, tickets.number AS ticket_number, tickets.title AS ticket_title, notifications.comment_id, notifications.actor_id, actor.name AS actor_name, actor.avatar_url AS actor_avatar_url, notifications.created_at, notifications.read_at").Order("notifications.created_at DESC, notifications.id DESC").Limit(limit).Offset(offset).Scan(&rows).Error
	return rows, total, translate(err)
}
func (r *notificationRepository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&models.Notification{}).Where("recipient_id = ? AND withdrawn_at IS NULL AND read_at IS NULL", userID).Count(&total).Error
	return total, translate(err)
}
func (r *notificationRepository) Read(ctx context.Context, userID, id string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.Notification{}).Where("id = ? AND recipient_id = ? AND withdrawn_at IS NULL", id, userID).Update("read_at", gorm.Expr("COALESCE(read_at, ?)", now))
	if result.Error != nil {
		return translate(result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}
func (r *notificationRepository) ReadAll(ctx context.Context, userID string, cutoff time.Time) error {
	return translate(r.db.WithContext(ctx).Model(&models.Notification{}).Where("recipient_id = ? AND withdrawn_at IS NULL AND read_at IS NULL AND created_at <= ?", userID, cutoff).Update("read_at", cutoff).Error)
}
