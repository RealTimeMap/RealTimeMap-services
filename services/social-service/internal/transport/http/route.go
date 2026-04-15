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
}
