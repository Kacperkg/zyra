package services

import (
	"context"
	"time"
	"zyra-api/internal/apperrors"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
)

type ListOptions struct {
	ClosureActor        string
	Page, Limit         int
	Sort, Order, Search string
	Filters             map[string]any
	CreatedAfter        *time.Time
	CreatedBefore       *time.Time
}
type Page struct {
	Items any   `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}

func (s *Service) List(ctx context.Context, kind string, opts ListOptions) (Page, error) {
	var fetch func(context.Context, repository.Query, ListOptions) (Page, error)
	allowed := map[string]bool{"created_at": true, "id": true}
	filterColumns := map[string]bool{}
	columns := []string{}
	switch kind {
	case "clients":
		fetch = pageReader[models.Client](s.Store.Clients())
		allowed["name"] = true
		columns = []string{"name"}
		filterColumns["archived"] = true
	case "databases":
		fetch = pageReader[models.Database](s.Store.Databases())
		allowed["name"] = true
		columns = []string{"name", "hostname"}
		filterColumns["client_id"] = true
		filterColumns["archived"] = true
	case "tickets":
		for _, v := range []string{"number", "check_type", "created_at", "client", "client_name", "database", "database_name", "assessment_type", "status"} {
			allowed[v] = true
		}
		for _, v := range []string{"number", "status", "check_type", "assessment_type", "client_id", "database_id", "closed_by"} {
			filterColumns[v] = true
		}
	case "assessments":
		fetch = pageReader[models.Assessment](s.Store.Assessments())
		allowed["received_at"] = true
		filterColumns["database_id"] = true
		filterColumns["result"] = true
		filterColumns["type"] = true
	case "users":
		filterColumns["status"] = true
		if status, ok := opts.Filters["status"]; ok && status != "active" && status != "retired" {
			return Page{}, invalid("invalid account status")
		}
		fetch = pageReader[models.User](s.Store.Users())
		columns = []string{"email", "name"}
	default:
		return Page{}, apperrors.ErrNotFound
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Limit == 0 {
		opts.Limit = 50
	}
	if opts.Limit < 1 || opts.Limit > 200 || opts.Page > 100000 {
		return Page{}, invalid("invalid pagination")
	}
	if opts.Sort == "" {
		opts.Sort = "created_at"
		if kind == "assessments" {
			opts.Sort = "received_at"
		}
	}
	if !allowed[opts.Sort] {
		return Page{}, invalid("unsupported sort field")
	}
	if opts.Order == "" {
		opts.Order = "desc"
	}
	if opts.Order != "asc" && opts.Order != "desc" {
		return Page{}, invalid("order must be asc or desc")
	}
	for k := range opts.Filters {
		if !filterColumns[k] {
			return Page{}, invalid("unsupported filter: " + k)
		}
	}
	if kind == "tickets" {
		rows, total, err := s.Store.Tickets().ListSummaries(ctx, repository.TicketListQuery{
			ClosureActor: opts.ClosureActor, Search: opts.Search, Filters: opts.Filters,
			CreatedAfter: opts.CreatedAfter, CreatedBefore: opts.CreatedBefore,
			Sort: opts.Sort, Order: opts.Order, Limit: opts.Limit, Offset: (opts.Page - 1) * opts.Limit,
		})
		if err != nil {
			return Page{}, err
		}
		return Page{Items: rows, Total: total, Page: opts.Page, Limit: opts.Limit}, nil
	}
	q := repository.Query{Where: opts.Filters, Search: opts.Search, SearchColumns: columns, Order: opts.Sort + " " + opts.Order + ", id " + opts.Order, Limit: opts.Limit, Offset: (opts.Page - 1) * opts.Limit}
	q.ClosureActor = opts.ClosureActor
	return fetch(ctx, q, opts)
}

type listRepository[T any] interface {
	Find(context.Context, *[]T, repository.Query) error
	Count(context.Context, repository.Query) (int64, error)
}

func pageReader[T any](repo listRepository[T]) func(context.Context, repository.Query, ListOptions) (Page, error) {
	return func(ctx context.Context, q repository.Query, opts ListOptions) (Page, error) {
		out := []T{}
		n, err := repo.Count(ctx, q)
		if err != nil {
			return Page{}, err
		}
		if err = repo.Find(ctx, &out, q); err != nil {
			return Page{}, err
		}
		return Page{out, n, opts.Page, opts.Limit}, nil
	}
}
