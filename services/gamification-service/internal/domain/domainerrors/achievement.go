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
	ErrAchievementIconRequired = func() error {
		return apperror.NewFieldValidationError(
			"icon",
			"field required",
			"value_error.icon.required",
			"",
		)
	}
	ErrAchievementIconInvalid = func(icon string) error {
		return apperror.NewFieldValidationError(
			"icon",
			"icon name must be at most 128 characters",
			"value_error.icon.invalid",
			icon,
		)
	}
)
