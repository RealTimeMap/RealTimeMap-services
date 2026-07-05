package mark

import (
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

var (
	ErrMarkNotFound = func(id uint) error {
		return apperror.NewNotFoundErrorByID("mark_action", id)
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
)
