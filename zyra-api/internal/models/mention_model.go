package models

// CommentMention is the current mention set, including historical recipients.
type CommentMention struct {
	CommentID   string `gorm:"primaryKey"`
	RecipientID string `gorm:"primaryKey"`
}

// CommentNotificationDelivery survives withdrawal, enforcing once-ever delivery.
type CommentNotificationDelivery struct {
	CommentID   string `gorm:"primaryKey"`
	RecipientID string `gorm:"primaryKey"`
}
