package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrProgressNotFount = func(userID uint) error {
		return apperror.NewNotFoundError("user_progress", "user_id", userID)
	}
)
