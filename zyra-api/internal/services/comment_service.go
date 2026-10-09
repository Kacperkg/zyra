package services

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type CommentInput struct {
	Comment       *string                `json:"comment,omitempty"`
	SchemaVersion int                    `json:"schema_version,omitempty"`
	Content       *models.CommentContent `json:"content,omitempty"`
	Revision      int                    `json:"revision,omitempty"`
}

var commentColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validateComment(in CommentInput) (*models.CommentContent, []string, error) {
	content := in.Content
	if in.Comment != nil {
		if content != nil || in.SchemaVersion != 0 {
			return nil, nil, invalid("comment and rich content are mutually exclusive")
		}
		content = &models.CommentContent{Blocks: []models.CommentBlock{{Type: "paragraph", Children: []models.CommentInline{{Type: "text", Text: *in.Comment}}}}}
	} else if in.SchemaVersion != 1 || content == nil {
		return nil, nil, invalid("schema_version 1 and content required")
	}
	raw, err := json.Marshal(content)
	if err != nil || len(raw) > 65536 {
		return nil, nil, invalid("content exceeds 64 KiB")
	}
	if len(content.Blocks) == 0 || len(content.Blocks) > 100 {
		return nil, nil, invalid("content must contain 1–100 blocks")
	}
	ids := []string{}
	seen := map[string]bool{}
	nodes := 0
	meaningful := false
	textSize := 0
	gifs := 0
	for _, b := range content.Blocks {
		switch b.Type {
		case "paragraph", "bullet_list", "ordered_list", "code_block":
		default:
			return nil, nil, invalid("unsupported block type")
		}
		if b.Align != "" && b.Align != "left" && b.Align != "center" && b.Align != "right" {
			return nil, nil, invalid("invalid alignment")
		}
		for _, n := range b.Children {
			nodes++
			if nodes > 1000 {
				return nil, nil, invalid("too many content nodes")
			}
			if n.Type == "mention" {
				if n.UserID == "" || n.Text != "" || n.Href != "" || n.Color != "" || n.Size != 0 || n.Bold || n.Italic || n.Underline || n.Strike || n.Src != "" || n.Alt != "" {
					return nil, nil, invalid("mention contains unsupported fields")
				}
				if !seen[n.UserID] {
					ids = append(ids, n.UserID)
					seen[n.UserID] = true
				}
				meaningful = true
			} else if n.Type == "text" {
				if n.UserID != "" || n.Src != "" || n.Alt != "" {
					return nil, nil, invalid("text cannot contain mention or GIF fields")
				}
				textSize += len(n.Text)
				if strings.TrimSpace(n.Text) != "" {
					meaningful = true
				}
				switch n.Size {
				case 0, 12, 14, 16, 18, 20, 24:
				default:
					return nil, nil, invalid("unsupported text size")
				}
				if n.Color != "" && !commentColor.MatchString(n.Color) {
					return nil, nil, invalid("invalid text colour")
				}
				if n.Href != "" {
					p, e := url.Parse(n.Href)
					if e != nil || len(n.Href) > 2048 || (p.Scheme != "https" && p.Scheme != "http") || p.Hostname() == "" || p.User != nil || p.Opaque != "" {
						return nil, nil, invalid("link must be an HTTP(S) URL without credentials")
					}
				}
			} else if n.Type == "gif" {
				if n.UserID != "" || n.Text != "" || n.Href != "" || n.Color != "" || n.Size != 0 || n.Bold || n.Italic || n.Underline || n.Strike {
					return nil, nil, invalid("GIF contains unsupported fields")
				}
				p, e := url.Parse(n.Src)
				if e != nil || len(n.Src) > 2048 || p.Scheme != "https" || p.Hostname() == "" || p.User != nil || p.Opaque != "" || len(n.Alt) > 200 {
					return nil, nil, invalid("GIF requires an HTTPS URL without credentials (maximum 2048 bytes) and optional description up to 200 bytes")
				}
				gifs++
				if gifs > 10 {
					return nil, nil, invalid("maximum 10 GIFs per comment")
				}
				meaningful = true
			} else {
				return nil, nil, invalid("unsupported inline type")
			}
		}
	}
	if !meaningful || textSize > 20000 || len(ids) > 20 {
		return nil, nil, invalid("meaningful comment required; maximum 20000 text bytes and 20 mentions")
	}
	return content, ids, nil
}

func commentText(content *models.CommentContent, users map[string]models.User) string {
	lines := []string{}
	for _, b := range content.Blocks {
		var line strings.Builder
		for _, n := range b.Children {
			if n.Type == "mention" {
				line.WriteString("@" + users[n.UserID].Name)
			} else if n.Type == "gif" {
				alt := strings.TrimSpace(n.Alt)
				if alt == "" {
					line.WriteString("[GIF]")
				} else {
					line.WriteString("[GIF: " + alt + "]")
				}
			} else {
				line.WriteString(n.Text)
			}
		}
		lines = append(lines, line.String())
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (s *Service) WriteComment(ctx context.Context, actor models.User, ticketID, commentID string, in CommentInput, closeTicket bool) (models.Ticket, error) {
	var ticket models.Ticket
	content, recipients, err := validateComment(in)
	if err != nil {
		return ticket, err
	}
	if commentID != "" && in.Revision < 1 {
		return ticket, invalid("revision required")
	}
	if commentID == "" && in.Revision != 0 {
		return ticket, invalid("revision is only accepted on edits")
	}
	err = s.Store.Transaction(ctx, func(tx repository.Store) error {
		users, e := tx.Users().LockIDs(ctx, append([]string{actor.ID}, recipients...))
		if e != nil {
			return e
		}
		fresh, ok := users[actor.ID]
		if !ok || fresh.Status != models.StatusActive {
			return apperrors.ErrUnauthorized
		}
		if e = validateActorSession(ctx, tx, fresh.ID, s.Now()); e != nil {
			return e
		}
		if e = tx.Tickets().Get(ctx, &ticket, ticketID, true); e != nil {
			return e
		}
		now := s.Now().UTC()
		var comment models.Comment
		previous := map[string]bool{}
		if commentID != "" {
			if e = tx.Comments().Get(ctx, &comment, commentID, true); e != nil {
				return e
			}
			if comment.TicketID != ticketID || comment.DeletedAt != nil {
				return apperrors.ErrNotFound
			}
			if comment.UserID != actor.ID {
				return apperrors.ErrForbidden
			}
			if comment.Revision != in.Revision {
				return apperrors.ErrConflict
			}
			old, e := tx.Mentions().RecipientIDs(ctx, commentID)
			if e != nil {
				return e
			}
			for _, id := range old {
				previous[id] = true
			}
			comment.Revision++
			comment.EditedAt = &now
		} else {
			comment = models.Comment{ID: auth.Random(), TicketID: ticketID, UserID: actor.ID, Revision: 1, CreatedAt: now}
		}
		for _, id := range recipients {
			u, exists := users[id]
			if !exists || (u.Status != models.StatusActive && !previous[id]) {
				return invalid("mention recipient is unavailable")
			}
		}
		comment.SchemaVersion = 1
		comment.Content = content
		comment.Text = commentText(content, users)
		if commentID == "" {
			if e = tx.Comments().Create(ctx, &comment); e != nil {
				return e
			}
			if e = tx.TicketEvents().Create(ctx, &models.TicketEvent{ID: comment.ID, TicketID: ticketID, UserID: actor.ID, Type: "comment", CommentID: comment.ID, CreatedAt: now}); e != nil {
				return e
			}
		} else if e = tx.Comments().Save(ctx, &comment); e != nil {
			return e
		}
		if e = tx.Mentions().Reconcile(ctx, comment.ID, ticketID, actor.ID, recipients, now); e != nil {
			return e
		}
		if closeTicket {
			if ticket.Status == "closed" {
				return apperrors.ErrConflict
			}
			ticket.Status = "closed"
			ticket.ClosedAt = &now
			ticket.ClosedBy = actor.ID
			if e = tx.TicketEvents().Create(ctx, &models.TicketEvent{ID: comment.ID + "-closure", TicketID: ticketID, UserID: actor.ID, Type: "close", CreatedAt: now}); e != nil {
				return e
			}
		}
		ticket.UpdatedAt = now
		return tx.Tickets().Save(ctx, &ticket)
	})
	return ticket, err
}

func (s *Service) DeleteComment(ctx context.Context, actor models.User, ticketID, id string, revision int) error {
	if revision < 1 {
		return invalid("revision required")
	}
	return s.Store.Transaction(ctx, func(tx repository.Store) error {
		var user models.User
		if err := tx.Users().Get(ctx, &user, actor.ID, true); err != nil {
			return err
		}
		if user.Status != models.StatusActive {
			return apperrors.ErrUnauthorized
		}
		if err := validateActorSession(ctx, tx, user.ID, s.Now()); err != nil {
			return err
		}
		var ticket models.Ticket
		if err := tx.Tickets().Get(ctx, &ticket, ticketID, true); err != nil {
			return err
		}
		var comment models.Comment
		if err := tx.Comments().Get(ctx, &comment, id, true); err != nil {
			return err
		}
		if comment.TicketID != ticketID || comment.DeletedAt != nil {
			return apperrors.ErrNotFound
		}
		if comment.UserID != actor.ID {
			return apperrors.ErrForbidden
		}
		if comment.Revision != revision {
			return apperrors.ErrConflict
		}
		now := s.Now().UTC()
		comment.DeletedAt = &now
		comment.Content = nil
		comment.Text = ""
		comment.Revision++
		if err := tx.Comments().Save(ctx, &comment); err != nil {
			return err
		}
		return tx.Mentions().WithdrawAll(ctx, id, now)
	})
}
