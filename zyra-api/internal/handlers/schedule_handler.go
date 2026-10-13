package handlers

import (
	"github.com/gin-gonic/gin"
	"zyra-api/internal/models"
	"zyra-api/internal/services"
)

func (h *Handler) Schedules(c *gin.Context) {
	if c.Request.Method == "GET" {
		v, e := h.Service.SourceSchedules(c.Request.Context(), c.Param("source_id"))
		respond(c, 200, v, e)
		return
	}
	var in struct {
		Schedules []models.Schedule `json:"schedules"`
		Weekdays  *[]string         `json:"weekdays"`
	}
	if !bind(c, &in) {
		return
	}
	v, e := h.Service.SaveSourceInput(c.Request.Context(), actor(c), "", c.Param("source_id"), services.EmailSourceInput{Schedules: &in.Schedules, Weekdays: in.Weekdays})
	respond(c, 200, v.Schedules, e)
}

func (h *Handler) EvaluateSchedules(c *gin.Context) {
	var in struct {
		Date string `json:"date"`
	}
	if !bind(c, &in) {
		return
	}
	n, e := h.Service.EvaluateSchedules(c.Request.Context(), actor(c), in.Date)
	respond(c, 200, gin.H{"created": n}, e)
}
