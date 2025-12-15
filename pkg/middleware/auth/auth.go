package auth

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		userNameStr := c.GetHeader("X-User-Name")
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"details": "Invalid X-User-ID",
			})
			return
		}
		if userNameStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"details": "Invalid X-User-Name",
			})
			return
		}
		c.Set(UserIDKey, userID)
		c.Set(UsernameKey, userNameStr)
		c.Next()
	}
}
