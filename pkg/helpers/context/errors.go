package context

import "errors"

var (
	ErrUserIDNotFound   = errors.New("user_id not found in context")
	ErrInvalidType      = errors.New("invalid type in context")
	ErrUserNamaNotFound = errors.New("username not found in context")
)
