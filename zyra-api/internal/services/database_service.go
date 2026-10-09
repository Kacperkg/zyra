package services

import (
	"context"
	"net"
	"strings"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/auth"
	"zyra-api/internal/models"
)

type DatabaseInput struct {
	ClientID *string `json:"client_id"`
	Name     *string `json:"name"`
	Hostname *string `json:"hostname"`
	IP       *string `json:"ip"`
	Notes    *string `json:"notes"`
}

func (s *Service) SaveDatabase(ctx context.Context, u models.User, id string, in DatabaseInput) (models.Database, error) {
	var result models.Database
	err := s.withActorWrite(ctx, u, requireConfigure, func(scoped *Service, fresh models.User) error {
		var err error
		result, err = scoped.saveDatabase(ctx, fresh, id, in)
		return err
	})
	return result, err
}
func (s *Service) saveDatabase(ctx context.Context, u models.User, id string, in DatabaseInput) (models.Database, error) {
	var d models.Database
	if err := requireConfigure(u); err != nil {
		return d, err
	}
	if id != "" {
		if err := s.Store.Databases().Get(ctx, &d, id, true); err != nil {
			return d, err
		}
		if d.Archived {
			return d, apperrors.ErrConflict
		}
	} else {
		d.ID = auth.Random()
		d.CreatedAt = s.Now().UTC()
	}
	if in.ClientID != nil {
		if id != "" && d.ClientID != *in.ClientID {
			return d, invalid("database client cannot be changed")
		}
		d.ClientID = *in.ClientID
	}
	if in.Name != nil {
		d.Name = strings.TrimSpace(*in.Name)
	}
	if in.Hostname != nil {
		d.Hostname = strings.TrimSpace(*in.Hostname)
	}
	if in.IP != nil {
		d.IP = strings.TrimSpace(*in.IP)
	}
	if in.Notes != nil {
		d.Notes = *in.Notes
	}
	if d.Name == "" || len(d.Name) > 200 {
		return d, invalid("name must contain 1–200 bytes")
	}
	if d.IP != "" && net.ParseIP(d.IP) == nil {
		return d, invalid("invalid IP address")
	}
	var c models.Client
	if err := s.Store.Clients().Get(ctx, &c, d.ClientID, false); err != nil {
		return d, err
	}
	if c.Archived {
		return d, apperrors.ErrConflict
	}
	if id == "" {
		return d, s.Store.Databases().Create(ctx, &d)
	}
	return d, s.Store.Databases().Save(ctx, &d)
}
