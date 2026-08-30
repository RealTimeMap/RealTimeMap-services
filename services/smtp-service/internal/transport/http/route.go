package http

import (
	nethttp "net/http"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

// queueStallThreshold — возраст старейшего письма, после которого очередь
// считается вставшей.
//
// Отправка не имеет обратной связи в интерфейсе: пользователь просто не
// получает письмо и молчит. Возраст очереди — единственный симптом, видимый
// до того, как начнутся жалобы.
const queueStallThreshold = 15 * time.Minute

func RegisterRoutes(g *gin.Engine, di *app.Container) {
	g.Use(middleware.TraceMiddleware())

	api := g.Group("/api/v2")

	handlers.NewEmailHandler(api, handlers.EmailHandlerDeps{
		Emails:  di.Emails,
		Events:  di.Events,
		Emailer: di.Emailer,
		KeySrv:  di.KeyService,
		Logger:  di.Logger,
	})
	handlers.NewApiKeyHandler(api, handlers.ApiKeyDeps{
		Service: di.KeyService,
		Logger:  di.Logger,
	})

	g.GET("/smtp/health", healthHandler(di))
}

// healthHandler проверяет базу и состояние очереди.
//
// SMTP-сервер намеренно не опрашивается: провайдеры считают регулярные
// подключения без отправки злоупотреблением.
func healthHandler(di *app.Container) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		sqlDB, err := di.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(nethttp.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"service": "smtp-service",
				"reason":  "database unavailable",
			})
			return
		}

		age, err := di.Emails.OldestQueuedAge(ctx, time.Now())
		if err != nil {
			c.JSON(nethttp.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"service": "smtp-service",
				"reason":  "queue check failed",
			})
			return
		}

		body := gin.H{
			"status":            "healthy",
			"service":           "smtp-service",
			"oldest_queued_sec": int64(age.Seconds()),
		}

		if age > queueStallThreshold {
			body["status"] = "degraded"
			body["reason"] = "queue is not draining"
			c.JSON(nethttp.StatusServiceUnavailable, body)
			return
		}

		c.JSON(nethttp.StatusOK, body)
	}
}
