package domainerrors

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var (
	XPRewardNotFoundError = func(id uint) error {
		return apperror.NewNotFoundErrorByID("xp_reward", id)
	}
)
