package domainerrors

import (
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

// Category validation domainerrors
var (
	ErrCategoryNameRequired = func() error {
		return apperror.NewRequiredError("categoryName")
	}

	ErrCategoryNameTooLong = func(name string) error {
		return apperror.NewTooLongError("categoryName", 64, name)
	}

	ErrCategoryAlreadyExists = func(name string) error {
		return apperror.NewAlreadyExistsError("categoryName", name)
	}

	ErrCategoryColorRequired = func() error {
		return apperror.NewRequiredError("color")
	}

	ErrCategoryColorInvalid = func(color string) error {
		return apperror.NewInvalidFormatError("color", "hex (#RRGGBB)", color)
	}

	ErrCategoryIconRequired = func() error {
		return apperror.NewRequiredError("icon")
	}

	ErrCategoryIconInvalid = func() error {
		return apperror.NewInvalidImageError("icon")
	}

	ErrCategoryIconMimeType = func(mimeType string) error {
		return apperror.NewInvalidMimeTypeError(
			"icon",
			[]string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"},
			mimeType,
		)
	}

	ErrCategoryNotFound = func(value any) error {
		return apperror.NewNotFoundError("category", "categoryId", value)
	}
)

// Business domain errors

var (
	ErrCannotDeleteActiveCategory = func(categoryName string, markCount int) error {
		return apperror.NewConflictError(
			"categoryId",
			fmt.Sprintf("cannot delete category '%s' with %d active marks", categoryName, markCount),
			nil,
		)
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
