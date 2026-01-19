package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrConfigNotFount = func(event string) error {
		return apperror.NewNotFoundError("config", "event", event)
	}

	ErrConfigNotActive = func(event string) error {
		return apperror.NewFieldValidationError("event", "eventConfig is not active", "value_error.category.inactive", event)
	}
)
