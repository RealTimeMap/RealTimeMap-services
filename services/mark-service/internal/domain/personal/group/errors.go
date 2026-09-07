package group

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

var (
	ErrAlreadyExistGroup = func(name string) error {
		return apperror.NewAlreadyExistsError("name", name)
	}

	ErrNotFoundGroup = func(val any) error {
		return apperror.NewNotFoundErrorByID("group", val)
	}
)
