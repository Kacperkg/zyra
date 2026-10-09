package handlers

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"zyra-api/internal/services"
)

func personalPages(c *gin.Context) (int, int, bool) {
	page, limit := 1, 50
	for _, key := range []string{"page", "limit"} {
		if raw := c.Query(key); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 {
				c.JSON(400, gin.H{"error": "invalid pagination"})
				return 0, 0, false
			}
			if key == "page" {
				page = value
			} else {
				limit = value
			}
		}
	}
	return page, limit, true
}
func (h *Handler) SavedTickets(c *gin.Context) {
	page, limit, ok := personalPages(c)
	if !ok {
		return
	}
	v, e := h.Service.SavedTickets(c.Request.Context(), actor(c), c.Query("q"), page, limit)
	respond(c, 200, v, e)
}
func (h *Handler) SaveTicket(c *gin.Context) {
	var in services.SavedTicketInput
	if !bind(c, &in) {
		return
	}
	v, e := h.Service.SaveTicket(c.Request.Context(), actor(c), c.Param("ticket_id"), in)
	respond(c, 200, v, e)
}
func (h *Handler) UnsaveTicket(c *gin.Context) {
	e := h.Service.UnsaveTicket(c.Request.Context(), actor(c), c.Param("ticket_id"))
	respond(c, 200, gin.H{"ok": true}, e)
}
