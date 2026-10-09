package services

import (
	"context"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type TicketDetail struct {
	Ticket       models.Ticket                 `json:"ticket"`
	Client       models.Client                 `json:"client"`
	Database     models.Database               `json:"database"`
	Similar      []repository.SimilarTicketRow `json:"similar"`
	Participants []models.User                 `json:"participants"`
	SavedTicket  *models.SavedTicket           `json:"saved_ticket"`
}

func (s *Service) TicketDetail(ctx context.Context, id string, viewerID ...string) (TicketDetail, error) {
	d := TicketDetail{Similar: []repository.SimilarTicketRow{}, Participants: []models.User{}}
	if err := s.Store.Tickets().Get(ctx, &d.Ticket, id, false); err != nil {
		return d, err
	}
	if err := s.Store.Clients().Get(ctx, &d.Client, d.Ticket.ClientID, false); err != nil {
		return d, err
	}
	if err := s.Store.Databases().Get(ctx, &d.Database, d.Ticket.DatabaseID, false); err != nil {
		return d, err
	}
	var err error
	d.Similar, err = s.Store.Tickets().FindSimilar(ctx, d.Ticket, 5)
	if err != nil {
		return d, err
	}
	participantIDs, err := s.Store.TicketEvents().ParticipantUserIDs(ctx, id)
	if err != nil {
		return d, err
	}
	d.Participants, err = s.Store.Users().FindByIDs(ctx, participantIDs)
	if err != nil {
		return d, err
	}
	if len(viewerID) > 0 {
		saved, found, e := s.Store.SavedTickets().Get(ctx, viewerID[0], id)
		if e != nil {
			return d, e
		}
		if found {
			d.SavedTicket = &saved
		}
	}
	return d, nil
}

type TicketEventPage struct {
	Items []models.TicketEvent `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

func (s *Service) TicketEvents(ctx context.Context, id string, page int) (TicketEventPage, error) {
	result := TicketEventPage{Items: []models.TicketEvent{}, Page: page, Limit: 50}
	var ticket models.Ticket
	if err := s.Store.Tickets().Get(ctx, &ticket, id, false); err != nil {
		return result, err
	}
	q := repository.Query{Where: map[string]any{"ticket_id": ticket.ID}, Order: "created_at asc, id asc", Limit: result.Limit, Offset: (page - 1) * result.Limit}
	var err error
	result.Total, err = s.Store.TicketEvents().Count(ctx, q)
	if err != nil {
		return result, err
	}
	err = s.Store.TicketEvents().Find(ctx, &result.Items, q)
	if err != nil {
		return result, err
	}
	ids := []string{}
	for _, event := range result.Items {
		if event.CommentID != "" {
			ids = append(ids, event.CommentID)
		}
	}
	comments, e := s.Store.Comments().FindByIDs(ctx, ids)
	if e != nil {
		return result, e
	}
	byID := map[string]models.Comment{}
	mentionIDs := []string{}
	seenMention := map[string]bool{}
	for _, comment := range comments {
		byID[comment.ID] = comment
		if comment.Content != nil {
			for _, block := range comment.Content.Blocks {
				for _, node := range block.Children {
					if node.Type == "mention" && !seenMention[node.UserID] {
						seenMention[node.UserID] = true
						mentionIDs = append(mentionIDs, node.UserID)
					}
				}
			}
		}
	}
	mentioned, e := s.Store.Users().FindByIDs(ctx, mentionIDs)
	if e != nil {
		return result, e
	}
	identities := map[string]models.CommentMentionUser{}
	for _, user := range mentioned {
		identities[user.ID] = models.CommentMentionUser{ID: user.ID, Name: user.Name, AvatarURL: user.AvatarURL}
	}
	for i := range result.Items {
		event := &result.Items[i]
		if comment, ok := byID[event.CommentID]; ok {
			event.Comment = comment.Text
			event.Content = comment.Content
			event.SchemaVersion = comment.SchemaVersion
			event.Revision = comment.Revision
			event.EditedAt = comment.EditedAt
			if comment.Content != nil {
				seen := map[string]bool{}
				for _, block := range comment.Content.Blocks {
					for _, node := range block.Children {
						if node.Type == "mention" && !seen[node.UserID] {
							seen[node.UserID] = true
							if identity, ok := identities[node.UserID]; ok {
								event.MentionUsers = append(event.MentionUsers, identity)
							}
						}
					}
				}
			}
		}
	}
	return result, err
}

func (s *Service) TicketAction(ctx context.Context, u models.User, id, action, comment string) (models.Ticket, error) {
	var t models.Ticket
	if action == "comment" || action == "comment_and_close" {
		return s.WriteComment(ctx, u, id, "", CommentInput{Comment: &comment}, action == "comment_and_close")
	}
	// Initial policy follows the product permission matrix: authenticated users may comment/close/reopen.
	if !u.Role.Valid() || u.Status != models.StatusActive {
		return t, apperrors.ErrForbidden
	}
	err := s.Store.Transaction(ctx, func(tx repository.Store) error {
		var fresh models.User
		if err := tx.Users().Get(ctx, &fresh, u.ID, true); err != nil {
			return err
		}
		if fresh.Status != models.StatusActive {
			return apperrors.ErrUnauthorized
		}
		if err := validateActorSession(ctx, tx, fresh.ID, s.Now()); err != nil {
			return err
		}
		if err := tx.Tickets().Get(ctx, &t, id, true); err != nil {
			return err
		}
		switch action {
		case "close", "comment_and_close":
			if t.Status == "closed" {
				return apperrors.ErrConflict
			}
			now := s.Now().UTC()
			t.Status = "closed"
			t.ClosedAt = &now
			t.ClosedBy = u.ID
		case "reopen":
			if t.Status != "closed" {
				return apperrors.ErrConflict
			}
			t.Status = "open"
			t.ClosedAt = nil
			t.ClosedBy = ""
		case "comment":
		default:
			return invalid("invalid ticket action")
		}
		t.UpdatedAt = s.Now().UTC()
		if err := tx.Tickets().Save(ctx, &t); err != nil {
			return err
		}
		return tx.TicketEvents().Create(ctx, &models.TicketEvent{ID: auth.Random(), TicketID: id, UserID: u.ID, Type: action, Comment: comment, CreatedAt: t.UpdatedAt})
	})
	return t, err
}
