package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ContentSooLong = func(inputLength int) error {
		return apperror.NewFieldValidationError("context", "content soo long", "value_error.content.max_length", inputLength)
	}

	CommentNotFound = func(id uint) error {
		return apperror.NewNotFoundError("comment", "id", id)
	}

	CommentMaxDepthReached = func() error {
		return apperror.NewConflictError("comment.depth", "reach max depth for reply", "")
	}

	EntityTypeNotAllowed = func(entity string) error {
		return apperror.NewFieldValidationError("entityType", "entity not allowed", "value_error.entityType", entity)
	}

	CommentIsDeleted = func() error {
		return apperror.NewConflictError("comment.status", "is deleted", "")
	}

	NotCommentOwner = func() error {
		return apperror.NewForbiddenError("you are not the owner")
	}
)
