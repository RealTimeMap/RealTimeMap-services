package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	SubscriptionNotFound = func(id uint) error {
		return apperror.NewNotFoundError("subscription", "target_id", id)
	}
	AlreadySubscribed = func(id uint) error {
		return apperror.NewConflictError("target_id", "already subscribed to user", id)
	}
	CantSubscribeYourSelf = func(id uint) error {
		return apperror.NewConflictError("target_id", "user can't subscribe to yourself", id)
	}
	SubscriptionBlocked = func() error {
		return apperror.NewForbiddenError("subscription is not allowed due to a block between users")
	}
)
