package chat

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrChatNotFound = func(id uint) error {
		return apperror.NewNotFoundErrorByID("chat", id)
	}
	ErrSelfDirectChat = func(id uint) error {
		return apperror.NewConflictError("user_id", "can't open self chat", id)
	}
	ErrLowMembers = func(min uint) error {
		return apperror.NewConflictError("peersIds", "can't open low members", min)

	}
	ErrDuplicateMembers = func(ids []uint) error {
		return apperror.NewConflictError("peersIds", "can't open duplicate members", ids)
	}
	ErrNotParticipant = func() error {
		return apperror.NewForbiddenError("You are not a participant in this chat.")
	}
	ErrEmptyMessage = func() error {
		return apperror.NewRequiredError("content")
	}
	ErrMessageTooLong = func(maxLen int, value string) error {
		return apperror.NewTooLongError("content", maxLen, value)
	}
)
