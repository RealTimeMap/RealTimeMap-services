package key

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ApiKeyUnavailable = func() error {
		return apperror.NewForbiddenError("API key unavailable")
	}
	ApiKeyRequired = func() error {
		return apperror.NewForbiddenError("API key required")
	}
	ApiKeyExpired = func() error {
		return apperror.NewForbiddenError("API key was expired")
	}
	ApiKeyNotFound = func() error {
		return apperror.NewNotFoundError("api_key", "X-Api-Key", "Not found")
	}
)
