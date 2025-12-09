package pagination

import "errors"

var (
	ErrInvalidPage      = errors.New("page must be greater than 0")
	ErrInvalidPageSize  = errors.New("page_size must be greater than 0")
	ErrPageSizeTooSmall = errors.New("page_size is below minimum allowed")
	ErrPageSizeTooLarge = errors.New("page_size exceeds maximum allowed")
)
