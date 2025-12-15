package context

import (
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/gin-gonic/gin"
)

// GetIsAdmin проверяет является ли данный пользователь админом
func GetIsAdmin(c *gin.Context) (bool, error) {
	value, exists := c.Get(auth.UserIsAdminKey)
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
	value, exists := c.Get(auth.UserIDKey)
	if !exists {
		return 0, ErrUserIDNotFound
	}

	userID, ok := value.(int)
	if !ok {
		return 0, ErrInvalidType
	}

	return userID, nil
}

func GetUserName(c *gin.Context) (string, error) {
	value, exists := c.Get(auth.UsernameKey)
	if !exists {
		return "", ErrUserNamaNotFound
	}
	userName, ok := value.(string)
	if !ok {
		return "", ErrInvalidType
	}
	return userName, nil
}

func GetUserInfo(c *gin.Context) (int, string, error) {
	userName, err := GetUserName(c)
	userID, err := GetUserID(c)
	if err != nil {
		return 0, "", err
	}
	return userID, userName, nil
}
