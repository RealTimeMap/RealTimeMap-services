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
)
