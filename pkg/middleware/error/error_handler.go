package error

import (
	"errors"
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HandleError - универсальный обработчик доменных ошибок
func HandleError(c *gin.Context, err error, logger *zap.Logger) {
	var domainErr apperror.DomainError
	traceID := context.GetTraceID(c)
	// Проверяем, является ли ошибка доменной
	if errors.As(err, &domainErr) {
		status := domainErr.HTTPStatus()

		// Для 500 ошибок логируем детали
		if status == http.StatusInternalServerError {
			logger.Error("internal error",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("TraceID", traceID),
			)
			c.AbortWithStatusJSON(status, gin.H{"error": "internal server error"})
			return
		}

		if status == http.StatusForbidden {
			c.AbortWithStatusJSON(status, gin.H{"error": "forbidden"})
			return
		}

		// Для клиентских ошибок возвращаем детали
		validationErrors := domainErr.ToValidation()
		if len(validationErrors) > 0 {
			c.AbortWithStatusJSON(status, validation.Response(validationErrors...))
		} else {
			c.AbortWithStatusJSON(status, gin.H{"error": err.Error()})
		}
		return
	}

	// Неизвестная ошибка - логируем и возвращаем 500
	logger.Error("unknown error",
		zap.Error(err),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.String("TraceID", traceID),
	)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
