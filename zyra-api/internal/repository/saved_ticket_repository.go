package repository

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
	"zyra-api/internal/models"
)

type SavedTicketRow struct {
	TicketSummaryRow
	PersonalTitle string    `json:"personal_title"`
	DisplayTitle  string    `json:"display_title"`
	SavedAt       time.Time `json:"saved_at"`
}
type SavedTicketRepository interface {
	Get(context.Context, string, string) (models.SavedTicket, bool, error)
	Put(context.Context, string, string, *string, time.Time) (models.SavedTicket, error)
	Delete(context.Context, string, string) error
	List(context.Context, string, string, int, int) ([]SavedTicketRow, int64, error)
}
type savedTicketRepository struct{ db *gorm.DB }

func NewSavedTicketRepository(db *gorm.DB) SavedTicketRepository { return &savedTicketRepository{db} }
func (s *store) SavedTickets() SavedTicketRepository             { return NewSavedTicketRepository(s.db) }
func (r *savedTicketRepository) Get(ctx context.Context, userID, ticketID string) (models.SavedTicket, bool, error) {
	var row models.SavedTicket
	err := r.db.WithContext(ctx).Where("user_id = ? AND ticket_id = ?", userID, ticketID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, false, nil
	}
	return row, err == nil, translate(err)
}
func (r *savedTicketRepository) Put(ctx context.Context, userID, ticketID string, title *string, now time.Time) (models.SavedTicket, error) {
	row := models.SavedTicket{UserID: userID, TicketID: ticketID, SavedAt: now}
	conflict := clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "ticket_id"}}, DoNothing: true}
	if title != nil {
		row.Title = *title
		conflict.DoNothing = false
		conflict.DoUpdates = clause.Assignments(map[string]any{"title": *title})
	}
	if err := r.db.WithContext(ctx).Clauses(conflict).Create(&row).Error; err != nil {
		return row, translate(err)
	}
	result, _, err := r.Get(ctx, userID, ticketID)
	return result, err
}
func (r *savedTicketRepository) Delete(ctx context.Context, userID, ticketID string) error {
	return translate(r.db.WithContext(ctx).Where("user_id = ? AND ticket_id = ?", userID, ticketID).Delete(&models.SavedTicket{}).Error)
}
func (r *savedTicketRepository) List(ctx context.Context, userID, search string, limit, offset int) ([]SavedTicketRow, int64, error) {
	rows := []SavedTicketRow{}
	db := r.db.WithContext(ctx).Table("saved_tickets").Joins("JOIN tickets ON tickets.id = saved_tickets.ticket_id").Joins("JOIN clients ON clients.id = tickets.client_id").Joins("JOIN databases ON databases.id = tickets.database_id").Where("saved_tickets.user_id = ?", userID)
	if search != "" {
		like := "%" + search + "%"
		db = db.Where("saved_tickets.title ILIKE ? OR tickets.title ILIKE ? OR clients.name ILIKE ? OR databases.name ILIKE ? OR CAST(tickets.number AS text) ILIKE ?", like, like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return rows, 0, translate(err)
	}
	err := db.Select(`tickets.id,tickets.number,tickets.title,tickets.status,tickets.check_type,tickets.assessment_type,tickets.client_id,clients.name AS client_name,tickets.database_id,databases.name AS database_name,tickets.created_at,tickets.closed_at,saved_tickets.title AS personal_title,COALESCE(NULLIF(saved_tickets.title,''),tickets.title) AS display_title,saved_tickets.saved_at`).Order("saved_tickets.saved_at DESC, saved_tickets.ticket_id DESC").Limit(limit).Offset(offset).Scan(&rows).Error
	return rows, total, translate(err)
}
