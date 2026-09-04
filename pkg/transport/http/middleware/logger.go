package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZapLogger заменяет стандартный gin.Logger(): пишет доступ-логи через zap,
// а не в собственном текстовом формате Gin. Благодаря этому у каждой записи
// есть поле level, и логи HTTP-слоя перестают попадать в Grafana как unknown.
//
// Уровень выбирается по статусу ответа: 5xx — error, 4xx — warn, остальное — info.
func ZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		status := c.Writer.Status()
		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", time.Since(start)),
			zap.Int("size", c.Writer.Size()),
		}
		if query != "" {
			fields = append(fields, zap.String("query", query))
		}
		if traceID, ok := c.Get("traceID"); ok {
			if id, isString := traceID.(string); isString && id != "" {
				fields = append(fields, zap.String("TraceID", id))
			}
		}
		// Ошибки, накопленные хендлерами через c.Error(...)
		if errs := c.Errors.ByType(gin.ErrorTypePrivate).String(); errs != "" {
			fields = append(fields, zap.String("errors", errs))
		}

		switch {
		case status >= 500:
			logger.Error("http request", fields...)
		case status >= 400:
			logger.Warn("http request", fields...)
		default:
			logger.Info("http request", fields...)
		}
	}
}

// ZapRecovery заменяет стандартный gin.Recovery(): паника логируется через zap
// со стектрейсом и уровнем error, клиенту уходит 500.
func ZapRecovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Стектрейс не добавляем вручную: логгер настроен через
				// WithStacktrace(ErrorLevel) и допишет его сам — иначе в JSON
				// окажется два ключа stacktrace.
				fields := []zap.Field{
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
				}
				if traceID, ok := c.Get("traceID"); ok {
					if id, isString := traceID.(string); isString && id != "" {
						fields = append(fields, zap.String("TraceID", id))
					}
				}
				logger.Error("panic recovered", fields...)

				c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
			}
		}()

		c.Next()
	}
}
