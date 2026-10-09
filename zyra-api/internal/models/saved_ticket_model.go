package models

import "time"

// SavedTicket belongs only to its user; retitling preserves SavedAt.
type SavedTicket struct {
	UserID   string    `json:"-" gorm:"primaryKey;index:idx_saved_user_time,priority:1"`
	TicketID string    `json:"ticket_id" gorm:"primaryKey;index:idx_saved_user_time,priority:3"`
	Title    string    `json:"personal_title"`
	SavedAt  time.Time `json:"saved_at" gorm:"index:idx_saved_user_time,priority:2"`
}
