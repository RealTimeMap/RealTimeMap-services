package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrPermissionDenied = func() error {
		return apperror.NewForbiddenError("forbidden")
	}
)
