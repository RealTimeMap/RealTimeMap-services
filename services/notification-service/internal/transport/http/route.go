package http

import (
	"github.com/gin-gonic/gin"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/transport/http/handlers"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2", auth.NotBanned())

	handlers.InitTokenHandler(api, handlers.TokenDeps{
		Service: container.TokenService,
		Logger:  container.Logger,
	})

	health := http.HealthHandler("notification-service", container.DB)
	g.GET("/notification/health", health)
}
