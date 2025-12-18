package context

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/gin-gonic/gin"
)

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

func GetUserInfo(c *gin.Context) (UserInput, error) {
	userName, err := GetUserName(c)
	if err != nil {
		return UserInput{}, err
	}
	userID, err := GetUserID(c)
	if err != nil {
		return UserInput{}, err
	}
	isAdmin, ok := c.Get(auth.UserIsAdminKey)
	if !ok {
		isAdmin = false
	}
	return NewUserInput(userID, userName, isAdmin.(bool)), nil

}
