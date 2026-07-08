package like

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	AlreadyReacted = func(userID uint) error {
		return apperror.NewConflictError("like", "user already like this object", userID)
	}
)
