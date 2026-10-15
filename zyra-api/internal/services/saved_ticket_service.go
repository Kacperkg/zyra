package services

import (
	"context"
	"strings"
	"zyra-api/internal/models"
)

type SavedTicketInput struct {
	Title *string `json:"title"`
}

func personalPagination(page, limit int) (int, int, error) {
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 50
	}
	if page < 1 || page > 1000000 || limit < 1 || limit > 200 {
		return 0, 0, invalid("page must be 1–1000000 and limit 1–200")
	}
	return page, limit, nil
}
func (s *Service) SavedTickets(ctx context.Context, actor models.User, q string, page, limit int) (Page, error) {
	page, limit, err := personalPagination(page, limit)
	result := Page{Page: page, Limit: limit}
	if err != nil {
		return result, err
	}
	if len(q) > 500 {
		return result, invalid("search exceeds 500 bytes")
	}
	items, total, err := s.Store.SavedTickets().List(ctx, actor.ID, strings.TrimSpace(q), limit, (page-1)*limit)
	result.Items = items
	result.Total = total
	return result, err
}
func (s *Service) SaveTicket(ctx context.Context, actor models.User, ticketID string, in SavedTicketInput) (models.SavedTicket, error) {
	var saved models.SavedTicket
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if len(title) > 200 {
			return saved, invalid("personal title exceeds 200 bytes")
		}
		in.Title = &title
	}
	err := s.withActorWrite(ctx, actor, requireParticipation, func(scoped *Service, fresh models.User) error {
		tx := scoped.Store
		var ticket models.Ticket
		if err := tx.Tickets().Get(ctx, &ticket, ticketID, false); err != nil {
			return err
		}
		var err error
		saved, err = tx.SavedTickets().Put(ctx, fresh.ID, ticket.ID, in.Title, s.Now().UTC())
		return err
	})
	return saved, err
}
func (s *Service) UnsaveTicket(ctx context.Context, actor models.User, ticketID string) error {
	return s.withActorWrite(ctx, actor, requireParticipation, func(scoped *Service, fresh models.User) error {
		return scoped.Store.SavedTickets().Delete(ctx, fresh.ID, ticketID)
	})
}
