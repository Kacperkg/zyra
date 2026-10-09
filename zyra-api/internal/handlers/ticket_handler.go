package handlers

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"zyra-api/internal/services"
)

func (h *Handler) Ticket(c *gin.Context) {
	v, e := h.Service.TicketDetail(c.Request.Context(), c.Param("id"), actor(c).ID)
	respond(c, 200, v, e)
}
func (h *Handler) Action(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if action == "close" || action == "reopen" {
			v, e := h.Service.TicketAction(c.Request.Context(), actor(c), c.Param("id"), action, "")
			respond(c, 200, v, e)
			return
		}
		var in services.CommentInput
		if !bind(c, &in) {
			return
		}
		v, e := h.Service.WriteComment(c.Request.Context(), actor(c), c.Param("id"), "", in, action == "comment_and_close")
		respond(c, 200, v, e)
	}
}

func (h *Handler) EditComment(c *gin.Context) {
	var in services.CommentInput
	if !bind(c, &in) {
		return
	}
	v, e := h.Service.WriteComment(c.Request.Context(), actor(c), c.Param("id"), c.Param("comment_id"), in, false)
	respond(c, 200, v, e)
}
func (h *Handler) DeleteComment(c *gin.Context) {
	revision, err := strconv.Atoi(c.Query("revision"))
	if err != nil || revision < 1 {
		c.JSON(400, gin.H{"error": "revision required"})
		return
	}
	e := h.Service.DeleteComment(c.Request.Context(), actor(c), c.Param("id"), c.Param("comment_id"), revision)
	respond(c, 200, gin.H{"deleted": true}, e)
}
