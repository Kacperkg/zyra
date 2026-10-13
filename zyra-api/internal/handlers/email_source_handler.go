package handlers

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"zyra-api/internal/services"
)

func (h *Handler) Source(c *gin.Context) {
	v, e := h.Service.EmailSource(c.Request.Context(), c.Param("source_id"))
	respond(c, 200, v, e)
}

func (h *Handler) DisableSource(c *gin.Context) {
	e := h.Service.DisableSource(c.Request.Context(), actor(c), c.Param("source_id"))
	respond(c, 200, gin.H{"enabled": false}, e)
}

func (h *Handler) Sources(c *gin.Context) {
	if c.Query("page") != "" || c.Query("limit") != "" {
		page, limit := 1, 50
		var err error
		if raw := c.Query("page"); raw != "" {
			page, err = strconv.Atoi(raw)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid pagination"})
				return
			}
		}
		if raw := c.Query("limit"); raw != "" {
			limit, err = strconv.Atoi(raw)
			if err != nil {
				c.JSON(400, gin.H{"error": "invalid pagination"})
				return
			}
		}
		v, e := h.Service.EmailSourcePage(c.Request.Context(), c.Param("id"), page, limit)
		respond(c, 200, v, e)
		return
	}
	v, e := h.Service.EmailSources(c.Request.Context(), c.Param("id"))
	respond(c, 200, v, e)
}
func (h *Handler) SaveSource(c *gin.Context) {
	var in services.EmailSourceInput
	if !bind(c, &in) {
		return
	}
	v, e := h.Service.SaveSourceInput(c.Request.Context(), actor(c), c.Param("id"), c.Param("source_id"), in)
	respond(c, 200, v, e)
}
