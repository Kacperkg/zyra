package models

import "time"

type Notification struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	RecipientID string     `json:"-" gorm:"index:idx_notification_recipient_time,priority:1;uniqueIndex:idx_notification_comment_recipient,priority:2"`
	ActorID     string     `json:"actor_id"`
	TicketID    string     `json:"ticket_id"`
	CommentID   string     `json:"comment_id" gorm:"uniqueIndex:idx_notification_comment_recipient,priority:1"`
	CreatedAt   time.Time  `json:"created_at" gorm:"index:idx_notification_recipient_time,priority:4"`
	ReadAt      *time.Time `json:"read_at" gorm:"index:idx_notification_recipient_time,priority:3"`
	WithdrawnAt *time.Time `json:"-" gorm:"index:idx_notification_recipient_time,priority:2"`
}
