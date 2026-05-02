package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2")

	handlers.RegisterLevelHandler(api, handlers.LevelDeps{
		Service: container.LevelService,
		Cache:   container.CacheStrategy,
		Logger:  container.Logger,
	})

	health := http.HealthHandler("gamification-service", container.DB)
	g.GET("/gamification/health", health)
}
