package personal

import (
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

var (
	ErrPhotosLimit = func(photos int) error {
		return apperror.NewFieldValidationError(
			"photos",
			fmt.Sprintf("maximum photos %d", maxPhotos),
			"value_error.photos.limit",
			photos,
		)
	}
	ErrGroupsRequired = func(ids []uint) error {
		return apperror.NewFieldValidationError(
			"groups",
			"minimum 1 groups required",
			"value_error.groups.required",
			ids,
		)
	}
	ErrStorageOperation = func(operation string, cause error) error {
		return apperror.WrapInternalError(
			fmt.Sprintf("storage %s failed", operation),
			cause,
		)
	}

	ErrAlreadyExistGroup = func(name string) error {
		return apperror.NewAlreadyExistsError("name", name)
	}

	ErrNotFoundGroup = func(val any) error {
		return apperror.NewNotFoundErrorByID("group", val)
	}
	ErrNotFoundMark = func(val any) error {
		return apperror.NewNotFoundErrorByID("personal_mark", val)
	}
	ErrOwnerShip = func() error {
		return apperror.NewForbiddenError("you are not owner")
	}
	ErrGroupColorInvalid = func(color string) error {
		return apperror.NewFieldValidationError(
			"color",
			"color must be a hex code like #RRGGBB",
			"value_error.color.invalid",
			color,
		)
	}
	ErrGroupIconInvalid = func(icon string) error {
		return apperror.NewFieldValidationError(
			"icon",
			fmt.Sprintf("icon name must be at most %d characters", maxGroupIconLen),
			"value_error.icon.invalid",
			icon,
		)
	}
	ErrGroupNotEmpty = func(marks int64) error {
		return apperror.NewFieldValidationError(
			"group",
			fmt.Sprintf("group still contains %d mark(s)", marks),
			"value_error.group.not_empty",
			marks,
		)
	}
)
