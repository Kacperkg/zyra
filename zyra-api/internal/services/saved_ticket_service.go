package services

import (
	"context"
	"strings"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type SavedTicketInput struct {
	Title *string `json:"title"`
}

func lockPersonalActor(ctx context.Context, tx repository.Store, actor models.User, now time.Time) error {
	var current models.User
	if err := tx.Users().Get(ctx, &current, actor.ID, true); err != nil {
		return err
	}
	if current.Status != models.StatusActive || !current.Role.Valid() {
		return apperrors.ErrForbidden
	}
	return validateActorSession(ctx, tx, current.ID, now)
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
	err := s.Store.Transaction(ctx, func(tx repository.Store) error {
		if err := lockPersonalActor(ctx, tx, actor, s.Now()); err != nil {
			return err
		}
		var ticket models.Ticket
		if err := tx.Tickets().Get(ctx, &ticket, ticketID, false); err != nil {
			return err
		}
		var err error
		saved, err = tx.SavedTickets().Put(ctx, actor.ID, ticket.ID, in.Title, s.Now().UTC())
		return err
	})
	return saved, err
}
func (s *Service) UnsaveTicket(ctx context.Context, actor models.User, ticketID string) error {
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		if err := lockPersonalActor(ctx, tx, actor, s.Now()); err != nil {
			return err
		}
		return tx.SavedTickets().Delete(ctx, actor.ID, ticketID)
	})
}
