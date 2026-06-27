package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	FriendRequestNotFound = func(id uint) error {
		return apperror.NewNotFoundError("friend_request", "friend_id", id)
	}
	FriendshipNotFound = func(id uint) error {
		return apperror.NewNotFoundError("friendship", "friend_id", id)
	}
	RequestAlreadyExists = func(id uint) error {
		return apperror.NewConflictError("friend_id", "friend request already exists", id)
	}
	AlreadyFriends = func(id uint) error {
		return apperror.NewConflictError("friend_id", "users are already friends", id)
	}
	CantFriendYourSelf = func(id uint) error {
		return apperror.NewConflictError("friend_id", "user can't friend yourself", id)
	}
	FriendshipBlocked = func() error {
		return apperror.NewForbiddenError("friendship is not allowed due to a block between users")
	}
)
