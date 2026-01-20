package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrLevelNotFount = func(level uint) error {
		return apperror.NewNotFoundError("levels", "level", level)
	}
)
