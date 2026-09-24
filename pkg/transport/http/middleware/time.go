package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

type customWriter struct {
	gin.ResponseWriter
	startTime time.Time
}

func (w *customWriter) WriteHeader(statusCode int) {
	latency := time.Since(w.startTime)
	w.Header().Set("X-Request-Time", latency.String())

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *customWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

func TimeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now().UTC()

		originalWriter := c.Writer
		c.Writer = &customWriter{ResponseWriter: originalWriter, startTime: start}

		c.Next()
	}
}
