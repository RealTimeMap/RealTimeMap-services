package domainerrors

import (
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

// Mark validation errors
var (
	ErrMarkNotFound = func(id int) error {
		return apperror.NewNotFoundErrorByID("mark", id)
	}
	ErrMarkNameRequired = func() error {
		return apperror.NewRequiredError("mark_name")
	}

	ErrMarkNameTooShort = func(name string) error {
		return apperror.NewFieldValidationError(
			"mark_name",
			"must be at least 3 characters",
			"value_error.any_str.min_length",
			name,
		)
	}

	ErrMarkNameTooLong = func(name string) error {
		return apperror.NewTooLongError("mark_name", 100, name)
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
			"start_at",
			fmt.Sprintf("cannot be more than %d days in the past", maxDays),
			"value_error.date.past_limit",
			nil,
		)
	}

	ErrStartAtTooFuture = func(maxDays int) error {
		return apperror.NewFieldValidationError(
			"start_at",
			fmt.Sprintf("cannot be more than %d days in the future", maxDays),
			"value_error.date.future_limit",
			nil,
		)
	}

	ErrPhotosRequired = func() error {
		return apperror.NewRequiredError("photos")
	}

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
			"category_id",
			"category is not active",
			"value_error.category.inactive",
			categoryId,
		)
	}
)

// Mark business errors

var (
	ErrDailyMarkLimitExceeded = func(limit int) error {
		return apperror.NewConflictError(
			"user_id",
			fmt.Sprintf("daily mark creation limit exceeded (%d marks per day)", limit),
			nil,
		)
	}
)
