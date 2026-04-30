package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2")

	// endpoints

	handlers.InitCategoryHandler(api, container.CategoryService, container.Logger)
	handlers.InitMarkHandler(api, container.MarkService, container.Logger)
	handlers.InitAdminMarkHandler(api, container.AdminMarkService, container.Logger)

	healthHandler := func(c *gin.Context) {
		// Проверка подключения к БД
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
			"service": "mark-service",
		})
	}

	// Support both GET and HEAD methods for health check
	api.GET("/mark/health", healthHandler)
	api.HEAD("/mark/health", healthHandler)

	// SOCKET

	socketApi := api.Group("/")
	socketApi.GET("/socket.io/*any", gin.WrapH(container.Socket.HttpHandler()))
	socketApi.POST("/socket.io/*any", gin.WrapH(container.Socket.HttpHandler()))

}
