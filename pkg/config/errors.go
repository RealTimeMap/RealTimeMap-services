package config

import "errors"

var (
	ErrConfigNotFound = errors.New("config not found")
	ErrInvalidConfig  = errors.New("invalid config format")
	ErrRequiredField  = errors.New("required field is empty")
)
