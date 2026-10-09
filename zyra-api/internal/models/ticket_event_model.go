package models

import (
	"time"
)

type TicketEvent struct {
	ID            string               `json:"id"`
	TicketID      string               `json:"ticket_id" gorm:"index:idx_event_timeline,priority:1;index:idx_event_closure,priority:3;index:idx_event_participant,priority:1"`
	UserID        string               `json:"user_id" gorm:"index:idx_event_closure,priority:1;index:idx_event_participant,priority:2"`
	Type          string               `json:"type" gorm:"index:idx_event_closure,priority:2"`
	Comment       string               `json:"comment,omitempty"`
	CreatedAt     time.Time            `json:"created_at" gorm:"index:idx_event_timeline,priority:2"`
	CommentID     string               `json:"comment_id,omitempty"`
	Content       *CommentContent      `json:"content,omitempty" gorm:"-"`
	SchemaVersion int                  `json:"schema_version,omitempty" gorm:"-"`
	Revision      int                  `json:"revision,omitempty" gorm:"-"`
	EditedAt      *time.Time           `json:"edited_at,omitempty" gorm:"-"`
	MentionUsers  []CommentMentionUser `json:"mention_users,omitempty" gorm:"-"`
}

// Display identities for referenced mentions, including retired historical users.
// Kept separate from writable content so callers cannot forge a mention label.
type CommentMentionUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}
