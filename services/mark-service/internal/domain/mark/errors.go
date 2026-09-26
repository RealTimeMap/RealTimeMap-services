package mark

import (
	"fmt"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

var (
	ErrMarkNotFound = func(id uint) error {
		return apperror.NewNotFoundErrorByID("mark_action", id)
	}
	// ErrNoActiveMarks — случайную метку выбрать не из чего: активных нет.
	ErrNoActiveMarks = func() error {
		return apperror.NewNotFoundError("mark_action", "status", "active")
	}
	ErrMarkNameTooShort = func(name string) error {
		return apperror.NewFieldValidationError(
			"markName",
			"must be at least 3 characters",
			"value_error.any_str.min_length",
			name,
		)
	}

	ErrMarkNameTooLong = func(name string) error {
		return apperror.NewTooLongError("markName", 100, name)
	}

	ErrInvalidDuration = func(duration int) error {
		return apperror.NewFieldValidationError(
			"duration",
			"must be one of: 12, 24, 36, 48 hours",
			"value_error.invalid_choice",
			duration,
		)
	}

	ErrStartAtTooOld = func(maxDays int) error {
		return apperror.NewFieldValidationError(
			"startAt",
			fmt.Sprintf("cannot be more than %d days in the past", maxDays),
			"value_error.date.past_limit",
			nil,
		)
	}

	ErrStartAtTooFuture = func(maxDays int) error {
		return apperror.NewFieldValidationError(
			"startAt",
			fmt.Sprintf("cannot be more than %d days in the future", maxDays),
			"value_error.date.future_limit",
			nil,
		)
	}
	ErrEndAtBeforeStart = func(endAt time.Time) error {
		return apperror.NewFieldValidationError(
			"endAt",
			"The end time cannot be earlier than the start time.",
			"value_error.date.conflict",
			endAt,
		)
	}
	ErrEndAtInPast = func(endAt time.Time) error {
		return apperror.NewFieldValidationError(
			"endAt",
			"The end time cannot be in the past.",
			"value_error.date.conflict",
			endAt,
		)
	}
	ErrEndAtMaxInFuture = func(maxDays int, endAt time.Time) error {
		return apperror.NewFieldValidationError(
			"endAt",
			fmt.Sprintf("The end date cannot be greater than %d days.", maxDays),
			"value_error.date.conflict",
			endAt,
		)
	}
	ErrMarkTTLTooShort = func(ttl int) error {
		return apperror.NewFieldValidationError(
			"endAt",
			fmt.Sprintf("The minimum total duration of the mark must be %d minutes.", ttl),
			"value_error.date.conflict",
			ttl,
		)
	}
)

var (
	ErrTooManyPhotos = func(count, max int) error {
		return apperror.NewFieldValidationError(
			"photos",
			fmt.Sprintf("maximum %d photos allowed, got %d", max, count),
			"value_error.list.max_length",
			count,
		)
	}

	ErrPhotoInvalidMimeType = func(index int, mimeType string) error {
		return apperror.NewFieldValidationError(
			fmt.Sprintf("photos[%d]", index),
			"must be image/jpeg, image/png, or image/webp",
			"value_error.mime_type",
			mimeType,
		)
	}

	ErrPhotoInvalidImage = func(index int) error {
		return apperror.NewFieldValidationError(
			fmt.Sprintf("photos[%d]", index),
			"file is not a valid image",
			"value_error.image",
			nil,
		)
	}

	ErrCategoryNotActive = func(categoryId int) error {
		return apperror.NewFieldValidationError(
			"categoryId",
			"category is not active",
			"value_error.category.inactive",
			categoryId,
		)
	}
	ErrLikeAlreadySet = func() error {
		return apperror.NewConflictError("like", "like for this mark_action already set", "")
	}
)

// Infrastructure domainerrors
var (
	ErrDatabaseQuery = func(operation string, cause error) error {
		return apperror.WrapInternalError(
			fmt.Sprintf("database %s failed", operation),
			cause,
		)
	}

	ErrStorageOperation = func(operation string, cause error) error {
		return apperror.WrapInternalError(
			fmt.Sprintf("storage %s failed", operation),
			cause,
		)
	}

	// ErrEndAtNotResolved — нарушен внутренний инвариант: validateDate обязан
	// проставить дефолтный EndAt, когда клиент его не прислал. Ошибка клиенту
	// не адресована, это сигнал о баге валидации.
	ErrEndAtNotResolved = func() error {
		return apperror.WrapInternalError("endAt was not resolved by validation", nil)
	}
)
