package http

import (
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
	healthHandler := func(c *gin.Context) {
		sqlDB, err := container.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(503, gin.H{
				"status":   "unhealthy",
				"database": "down",
			})
			return
		}

		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "social-service",
		})
	}
	api.GET("/social/health", healthHandler)
}
