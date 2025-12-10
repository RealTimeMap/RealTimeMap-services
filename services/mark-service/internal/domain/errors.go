package domain

import (
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

// Category validation errors
var (
	ErrCategoryNameRequired = func() error {
		return apperror.NewRequiredError("category_name")
	}

	ErrCategoryNameTooLong = func(name string) error {
		return apperror.NewTooLongError("category_name", 64, name)
	}

	ErrCategoryAlreadyExists = func(name string) error {
		return apperror.NewAlreadyExistsError("category_name", name)
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
		return apperror.NewNotFoundError("category", value)
	}
)

// Business errors
var (
	ErrCannotDeleteActiveCategory = func(categoryName string, markCount int) error {
		return apperror.NewConflictError(
			"category_id",
			fmt.Sprintf("cannot delete category '%s' with %d active marks", categoryName, markCount),
			nil,
		)
	}
)

// Infrastructure errors
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
