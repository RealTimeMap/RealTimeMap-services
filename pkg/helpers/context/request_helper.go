package context

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// GetIsAdmin проверяет является ли данный пользователь админом
func GetIsAdmin(c *gin.Context) (bool, error) {
	value, exists := c.Get(UserAdmin)
	if !exists {
		return false, errors.New("is_admin not found")
	}

	isAdmin, ok := value.(bool)
	if !ok {
		return false, ErrInvalidType
	}

	return isAdmin, nil
}

// GetUserID получение user_id из заголовков
func GetUserID(c *gin.Context) (int, error) {
	value, exists := c.Get(UserID)
	if !exists {
		return 0, ErrUserIDNotFound
	}

	userID, ok := value.(int)
	if !ok {
		return 0, ErrInvalidType
	}

	return userID, nil
}
