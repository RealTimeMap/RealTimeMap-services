package auth

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		isAdminStr := c.GetHeader("X-User-Admin")

		userID, err := strconv.Atoi(userIDStr)
		if err != nil || userID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":  "unauthorized",
				"detail": "valid X-User-ID header is required",
			})
			return
		}

		isAdmin, err := strconv.ParseBool(isAdminStr)
		if err != nil || !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":  "forbidden",
				"detail": "admin access required",
			})
			return
		}

		// Сохраняем типизированные значения
		c.Set("user_id", userID)
		c.Set("is_admin", isAdmin)
		c.Next()
	}
}
