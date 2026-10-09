package services

import (
	"context"
	"strings"
	"unicode/utf8"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

func (s *Service) MentionOptions(ctx context.Context, q string, limit int) ([]repository.MentionOption, error) {
	q = strings.TrimSpace(q)
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 20 {
		return nil, invalid("mention limit must be 1–20")
	}
	if len(q) > 200 {
		return nil, invalid("mention search exceeds 200 bytes")
	}
	if utf8.RuneCountInString(q) < 2 {
		return []repository.MentionOption{}, nil
	}
	return s.Store.Mentions().Search(ctx, q, limit)
}
func (s *Service) Notifications(ctx context.Context, actor models.User, unread bool, page, limit int) (Page, error) {
	page, limit, err := personalPagination(page, limit)
	result := Page{Page: page, Limit: limit}
	if err != nil {
		return result, err
	}
	items, total, err := s.Store.Notifications().List(ctx, actor.ID, unread, limit, (page-1)*limit)
	result.Items = items
	result.Total = total
	return result, err
}
func (s *Service) UnreadNotifications(ctx context.Context, actor models.User) (int64, error) {
	return s.Store.Notifications().UnreadCount(ctx, actor.ID)
}
func (s *Service) ReadNotification(ctx context.Context, actor models.User, id string) error {
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		if err := lockPersonalActor(ctx, tx, actor, s.Now()); err != nil {
			return err
		}
		return tx.Notifications().Read(ctx, actor.ID, id, s.Now().UTC())
	})
}
func (s *Service) ReadAllNotifications(ctx context.Context, actor models.User) error {
	cutoff := s.Now().UTC()
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		if err := lockPersonalActor(ctx, tx, actor, s.Now()); err != nil {
			return err
		}
		return tx.Notifications().ReadAll(ctx, actor.ID, cutoff)
	})
}
