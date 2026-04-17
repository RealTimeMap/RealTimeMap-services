package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	BlockedUserNotFound = func(id uint) error {
		return apperror.NewNotFoundError("blocked_user", "blocked_user_id", id)
	}
	UserAlreadyBlocked = func(id uint) error {
		return apperror.NewConflictError("blocked_user_id", "user already blocked", id)
	}
	CantBlockYourSelf = func(id uint) error {
		return apperror.NewConflictError("blocked_user_id", "user can't block yourself", id)
	}
)
