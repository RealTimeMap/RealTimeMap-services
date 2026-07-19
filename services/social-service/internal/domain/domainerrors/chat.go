package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ChatAlreadyExist = func() error {
		return apperror.NewConflictError("user_id", "chat already exist", nil)
	}
)
