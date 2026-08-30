package middleware

import (
	"net/http"
	"strings"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/key"
	"github.com/gin-gonic/gin"
)

const apiHeader = "X-Api-Key"

func ApiKeyRequiredMiddleware(srv *key.ApiKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(apiHeader)

		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
				"error":   "api key required",
			})
			return
		}

		if !strings.HasPrefix(raw, key.Prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
				"error":   "api key required",
			})
			return
		}
		if srv.ValidateToken(c.Request.Context(), raw) != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
				"error":   "api key required",
			})
			return
		}
		c.Next()
	}
}
