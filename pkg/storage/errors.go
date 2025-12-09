package storage

import "errors"

var (
	ErrInvalidCategory = errors.New("invalid category")
	ErrFileNotFound    = errors.New("file not found")
	ErrUploadFailed    = errors.New("upload failed")
	ErrInvalidMimeType = errors.New("invalid mime type")
	ErrFileTooLarge    = errors.New("file too large")
)
