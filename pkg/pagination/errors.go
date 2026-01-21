package pagination

import "errors"

var (
	ErrInvalidPage      = errors.New("page must be greater than 0")
	ErrInvalidPageSize  = errors.New("pageSize must be greater than 0")
	ErrPageSizeTooSmall = errors.New("pageSize is below minimum allowed")
	ErrPageSizeTooLarge = errors.New("pageSize exceeds maximum allowed")
)
