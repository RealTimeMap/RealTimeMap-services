package middleware

import (
	"context"
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

type ExistCheckFn func(ctx context.Context, id uint) (bool, error)

func Exist(check ExistCheckFn, log *zap.Logger, paramKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := c.Param(paramKey)
		id, err := strconv.ParseUint(param, 0, 64)
		if err != nil {
			err = apperror.NewFieldValidationError(paramKey, "must be a number", "value_error", param)
			HandleError(c, err, log)
			return
		}
		ok, err := check(c.Request.Context(), uint(id))
		if err != nil {
			HandleError(c, err, log)
			return
		}

		if !ok {
			err = apperror.NewNotFoundError("id not found", paramKey, param)
			HandleError(c, err, log)
			return
		}
		c.Next()

	}
}
