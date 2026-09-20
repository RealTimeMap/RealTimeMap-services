package token

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	ErrAlreadyExist = func(val string) error {
		return apperror.NewAlreadyExistsError("token", val)
	}
	ErrNotFoundUserToken = func(userID uint) error {
		return apperror.NewNotFoundError("user_devices", "user_id", userID)
	}
	ErrNotFoundDevice = func(deviceID string) error {
		return apperror.NewNotFoundError("user_devices", "device_id", deviceID)
	}
	ErrInvalidPlatform = func(val string) error {
		return apperror.NewInvalidFormatError("platform", "android|ios|web", val)
	}
	ErrUnknownKind = func(val string) error {
		return apperror.NewInvalidFormatError("kind", "chat|comment|subscriber", val)
	}
)
