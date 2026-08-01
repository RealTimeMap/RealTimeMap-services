package bug

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrBugTagUnavailable = func(tag string) error {
		return apperror.NewFieldValidationError("tag", "tag is not allowed", "value_error", tag)
	}
	ErrBugStatusUnavailable = func(status string) error {
		return apperror.NewFieldValidationError("tag", "status is not allowed", "value_error", status)
	}
)
