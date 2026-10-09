package repository

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
)

type MentionOption struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}
type MentionRepository interface {
	Search(context.Context, string, int) ([]MentionOption, error)
	RecipientIDs(context.Context, string) ([]string, error)
	Reconcile(context.Context, string, string, string, []string, time.Time) error
	WithdrawAll(context.Context, string, time.Time) error
}
type mentionRepository struct{ db *gorm.DB }

func NewMentionRepository(db *gorm.DB) MentionRepository { return &mentionRepository{db} }
func (s *store) Mentions() MentionRepository             { return NewMentionRepository(s.db) }
func (r *mentionRepository) Search(ctx context.Context, q string, limit int) ([]MentionOption, error) {
	rows := []MentionOption{}
	err := r.db.WithContext(ctx).Table("users").Select("id,name,avatar_url").Where("status = ? AND name ILIKE ?", "active", "%"+q+"%").Order("name ASC,id ASC").Limit(limit).Scan(&rows).Error
	return rows, translate(err)
}
func (r *mentionRepository) RecipientIDs(ctx context.Context, id string) ([]string, error) {
	ids := []string{}
	err := r.db.WithContext(ctx).Model(&models.CommentMention{}).Where("comment_id = ?", id).Order("recipient_id ASC").Pluck("recipient_id", &ids).Error
	return ids, translate(err)
}

// Reconcile must run in the comment transaction after its row and relevant users
// are locked/validated. The caller supplies the server-extracted mention set.
func (r *mentionRepository) Reconcile(ctx context.Context, commentID, ticketID, actorID string, recipients []string, now time.Time) error {
	db := r.db.WithContext(ctx)
	keep := []string{}
	seen := map[string]bool{}
	for _, id := range recipients {
		if !seen[id] {
			seen[id] = true
			keep = append(keep, id)
		}
	}
	removed := db.Model(&models.Notification{}).Where("comment_id = ? AND withdrawn_at IS NULL", commentID)
	if len(keep) > 0 {
		removed = removed.Where("recipient_id NOT IN ?", keep)
	}
	if err := removed.Update("withdrawn_at", now).Error; err != nil {
		return translate(err)
	}
	if err := db.Where("comment_id = ?", commentID).Delete(&models.CommentMention{}).Error; err != nil {
		return translate(err)
	}
	for _, recipientID := range keep {
		if err := db.Create(&models.CommentMention{CommentID: commentID, RecipientID: recipientID}).Error; err != nil {
			return translate(err)
		}
		if recipientID == actorID {
			continue
		}
		delivery := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.CommentNotificationDelivery{CommentID: commentID, RecipientID: recipientID})
		if delivery.Error != nil {
			return translate(delivery.Error)
		}
		if delivery.RowsAffected == 0 {
			continue
		}
		// Historical retired mentions can remain in the comment, but must not
		// generate a new notification. The service validates newly added IDs.
		var eligible int64
		if err := db.Model(&models.User{}).Where("id = ? AND status = ?", recipientID, "active").Count(&eligible).Error; err != nil {
			return translate(err)
		}
		if eligible == 0 {
			continue
		}
		if err := db.Create(&models.Notification{ID: auth.Random(), RecipientID: recipientID, ActorID: actorID, TicketID: ticketID, CommentID: commentID, CreatedAt: now}).Error; err != nil {
			return translate(err)
		}
	}
	return nil
}
func (r *mentionRepository) WithdrawAll(ctx context.Context, id string, now time.Time) error {
	db := r.db.WithContext(ctx)
	if err := db.Model(&models.Notification{}).Where("comment_id = ? AND withdrawn_at IS NULL", id).Update("withdrawn_at", now).Error; err != nil {
		return translate(err)
	}
	return translate(db.Where("comment_id = ?", id).Delete(&models.CommentMention{}).Error)
}
