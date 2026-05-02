package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2")

	handlers.RegisterProfileHandler(api, handlers.ProfileDeps{
		Service: container.ProfileService,
		Logger:  container.Logger,
	})
	handlers.RegisterBlockedUserHandler(api, handlers.BlockedDeps{
		Service: container.BlockedUserService,
		Logger:  container.Logger,
	})

	health := http.HealthHandler("social-service", container.DB)
	g.GET("/social/health", health)
}
