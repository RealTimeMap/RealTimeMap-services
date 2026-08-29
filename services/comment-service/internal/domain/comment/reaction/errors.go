package reaction

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	AlreadyReacted = func(userID uint) error {
		return apperror.NewConflictError("reaction", "user already liked this comment", userID)
	}
)
