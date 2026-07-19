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

// AuthOptional проставляет userID/userName в контекст, если заголовки присутствуют,
// но не прерывает запрос при их отсутствии. Используется для эндпоинтов,
// доступных и анонимно, где данные пользователя лишь обогащают ответ.
func AuthOptional() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		userNameStr := c.GetHeader("X-User-Name")

		if userID, err := strconv.Atoi(userIDStr); err == nil && userNameStr != "" {
			c.Set(UserIDKey, userID)
			c.Set(UsernameKey, userNameStr)
		}
		c.Next()
	}
}
