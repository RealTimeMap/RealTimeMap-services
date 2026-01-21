package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrConfigNotFount = func(event string) error {
		return apperror.NewNotFoundError("config", "event", event)
	}

	ErrConfigNotActive = func(event string) error {
		return apperror.NewFieldValidationError("event", "eventConfig is not active", "value_error.event.inactive", event)
	}

	ErrDailyLimitReached = func(limit uint) error {
		return apperror.NewFieldValidationError("dailyLimit", "daily limit reached", "value_error.dailyLimit.reached", limit)
	}
	ErrAlreadyEarnedForSource = func() error {
		return apperror.NewFieldValidationError("sourceID", "xp already earned for this source", "value_error.source_id.earned", "")
	}
)
