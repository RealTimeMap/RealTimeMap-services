package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	AchievementAlreadyExistError = func(code string) error {
		return apperror.NewAlreadyExistsError("code", code)
	}
	AchievementNotFoundError = func(field string, val interface{}) error {
		return apperror.NewNotFoundError("achievement", field, val)
	}
	AchievementAlreadyUnlockedError = func(id uint) error {
		return apperror.NewConflictError("achievement", "achievement already unlocked", id)
	}
)
