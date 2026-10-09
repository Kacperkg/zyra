package models

import "time"

// Content is editor-neutral data, never trusted HTML. Render text with escaping.
type CommentContent struct {
	Blocks []CommentBlock `json:"blocks"`
}
type CommentBlock struct {
	Type     string          `json:"type"`
	Align    string          `json:"align,omitempty"`
	Children []CommentInline `json:"children"`
}
type CommentInline struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Bold      bool   `json:"bold,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Underline bool   `json:"underline,omitempty"`
	Strike    bool   `json:"strike,omitempty"`
	Size      int    `json:"size,omitempty"`
	Color     string `json:"color,omitempty"`
	Href      string `json:"href,omitempty"`
	Src       string `json:"src,omitempty"`
	Alt       string `json:"alt,omitempty"`
}
type Comment struct {
	ID            string          `json:"id" gorm:"primaryKey"`
	TicketID      string          `json:"ticket_id" gorm:"index"`
	UserID        string          `json:"user_id"`
	SchemaVersion int             `json:"schema_version"`
	Content       *CommentContent `json:"content" gorm:"serializer:json;type:jsonb"`
	Text          string          `json:"text"`
	Revision      int             `json:"revision"`
	CreatedAt     time.Time       `json:"created_at"`
	EditedAt      *time.Time      `json:"edited_at,omitempty"`
	DeletedAt     *time.Time      `json:"-"`
}
