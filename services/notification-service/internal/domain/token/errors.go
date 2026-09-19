package token

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

var ErrAlreadyExist = func(val string) error {
	return apperror.NewAlreadyExistsError("token", val)
}
