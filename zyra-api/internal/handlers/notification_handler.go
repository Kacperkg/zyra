package handlers

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

func (h *Handler) MentionOptions(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			c.JSON(400, gin.H{"error": "invalid mention limit"})
			return
		}
		limit = n
	}
	v, e := h.Service.MentionOptions(c.Request.Context(), c.Query("q"), limit)
	respond(c, 200, gin.H{"items": v}, e)
}
func (h *Handler) Notifications(c *gin.Context) {
	page, limit, ok := personalPages(c)
	if !ok {
		return
	}
	unread := false
	if raw := c.Query("unread"); raw != "" {
		if raw != "true" && raw != "false" {
			c.JSON(400, gin.H{"error": "unread must be true or false"})
			return
		}
		unread = raw == "true"
	}
	v, e := h.Service.Notifications(c.Request.Context(), actor(c), unread, page, limit)
	respond(c, 200, v, e)
}
func (h *Handler) UnreadNotifications(c *gin.Context) {
	v, e := h.Service.UnreadNotifications(c.Request.Context(), actor(c))
	respond(c, 200, gin.H{"count": v}, e)
}
func (h *Handler) ReadNotification(c *gin.Context) {
	e := h.Service.ReadNotification(c.Request.Context(), actor(c), c.Param("id"))
	respond(c, 200, gin.H{"ok": true}, e)
}
func (h *Handler) ReadAllNotifications(c *gin.Context) {
	e := h.Service.ReadAllNotifications(c.Request.Context(), actor(c))
	respond(c, 200, gin.H{"ok": true}, e)
}
