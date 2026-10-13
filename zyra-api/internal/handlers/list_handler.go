package handlers

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"time"
	"zyra-api/internal/services"
)

func (h *Handler) List(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		opts := services.ListOptions{Sort: c.Query("sort"), Order: c.Query("order"), Search: c.Query("q"), Filters: map[string]any{}}
		for _, key := range []string{"page", "limit"} {
			if raw := c.Query(key); raw != "" {
				n, e := strconv.Atoi(raw)
				if e != nil || n < 1 {
					c.JSON(400, gin.H{"error": "invalid pagination"})
					return
				}
				if key == "page" {
					opts.Page = n
				} else {
					opts.Limit = n
				}
			}
		}
		for _, key := range []string{"engine", "assessment_id", "client_id", "database_id", "check_type", "assessment_type", "status", "result", "type", "closed_by", "number"} {
			if v := c.Query(key); v != "" {
				opts.Filters[key] = v
			}
		}
		if kind == "tickets" {
			var ok bool
			if opts.CreatedAfter, ok = ticketBoundary(c, "created_from", false); !ok {
				return
			}
			if opts.CreatedBefore, ok = ticketBoundary(c, "created_to", true); !ok {
				return
			}
		}
		if kind == "clients" || kind == "databases" {
			opts.Filters["archived"] = c.Query("archived") == "true"
		}
		if c.Param("id") != "" {
			if kind == "assessments" {
				opts.Filters["database_id"] = c.Param("id")
			}
			if kind == "tickets" {
				opts.ClosureActor = c.Param("id")
			}
		}
		v, e := h.Service.List(c.Request.Context(), kind, opts)
		respond(c, 200, v, e)
	}
}

func ticketBoundary(c *gin.Context, key string, endOfDate bool) (*time.Time, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		value = value.UTC()
		return &value, true
	}
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		c.JSON(400, gin.H{"error": key + " must be YYYY-MM-DD or RFC3339"})
		return nil, false
	}
	if endOfDate {
		value = value.AddDate(0, 0, 1)
	}
	return &value, true
}
