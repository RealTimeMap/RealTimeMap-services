package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ProfileNotFound = func(id uint) error {
		return apperror.NewNotFoundError("profile", "user_id", id)
	}
	ProfileAlreadyExists = func(id uint) error {
		return apperror.NewConflictError("profile", "user_id", id)
	}
	TagAlreadyTaken = func(tag string) error {
		return apperror.NewConflictError("profile", "tag", tag)
	}
	ProgressServiceUnavailable = func(err error) error {
		return apperror.NewServiceUnavailableError("gamification-service", err)
	}
	MarkServiceUnavailable = func(err error) error {
		return apperror.NewServiceUnavailableError("mark-service", err)
	}
)
