package token

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrAlreadyExist = func(val string) error {
		return apperror.NewAlreadyExistsError("token", val)
	}
	ErrNotFoundUserToken = func(userID uint) error {
		return apperror.NewNotFoundError("user_notification_toknes", "user_id", userID)
	}
)
