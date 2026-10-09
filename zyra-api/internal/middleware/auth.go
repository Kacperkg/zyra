package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"zyra-api/internal/models"
	"zyra-api/internal/services"
)

func Auth(s *services.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		u, sid, err := s.Authenticate(c.Request.Context(), parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
			return
		}
		c.Set("user", u)
		c.Set("session_id", sid)
		c.Request = c.Request.WithContext(services.WithSessionContext(c.Request.Context(), sid))
		c.Next()
	}
}
func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, _ := c.Get("user")
		u, ok := v.(models.User)
		if !ok || !u.Role.CanAdminister() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		c.Next()
	}
}
