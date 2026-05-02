package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
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

	// Health
	health := http.HealthHandler("mark-service", container.DB)
	g.GET("/mark/health", health)

	// SOCKET

	g.GET("/socket.io/*any", gin.WrapH(container.Socket.HttpHandler()))
	g.POST("/socket.io/*any", gin.WrapH(container.Socket.HttpHandler()))

}
