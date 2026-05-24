package middleware

import (
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
)

func ParsePathParams(c *gin.Context, key string) (uint, error) {
	param := c.Param(key)

	value, err := strconv.ParseUint(param, 10, 64)
	if err != nil {
		return 0, apperror.NewFieldValidationError(key, key+" must be a number", "value_error", param)
	}
	return uint(value), nil
}
