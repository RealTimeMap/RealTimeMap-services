package attraction

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var ErrAlreadyExist = func(slug string) error {
	return apperror.NewConflictError("slug", "attraction already exist", slug)
}

var ErrNotFound = func(id any) error {
	return apperror.NewNotFoundErrorByID("attraction", id)
}

var (
	ErrRequired = func(field string) apperror.DomainError {
		return apperror.NewRequiredError(field)
	}

	ErrTooLong = func(field string, max int, value string) apperror.DomainError {
		return apperror.NewTooLongError(field, max, value)
	}

	ErrSlugInvalid = func(slug string) apperror.DomainError {
		return apperror.NewInvalidFormatError("slug", "lowercase latin, digits and '-'", slug)
	}

	ErrGeomInvalid = func(lon, lat float64) apperror.DomainError {
		return apperror.NewInvalidFormatError("geom", "lon [-180..180], lat [-90..90]", []float64{lon, lat})
	}

	ErrStatusInvalid = func(value string) apperror.DomainError {
		return apperror.NewInvalidFormatError("status", "draft|published|archived", value)
	}

	ErrBlockTypeInvalid = func(field, value string) apperror.DomainError {
		return apperror.NewInvalidFormatError(field, "attribute|static", value)
	}

	ErrDurationInvalid = func(field string) apperror.DomainError {
		return apperror.NewFieldValidationError(field, "must be greater than 0", "value_error.number.not_gt", 0)
	}

	ErrPositionDuplicate = func(field string, position uint) apperror.DomainError {
		return apperror.NewFieldValidationError(field, "duplicate position", "value_error.duplicate", position)
	}
)
