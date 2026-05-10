package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2")

	profile := api.Group("/profile")
	// Основные роуты под профиль
	handlers.RegisterProfileHandler(profile, handlers.ProfileDeps{
		Service: container.ProfileService,
		Logger:  container.Logger,
	})
	// Вспомогательные роуты для статистики профиля
	handlers.RegisterStatHandler(profile, handlers.StatDeps{
		ProfileRepo: container.ProfileRepo,
		Logger:      container.Logger,
		Service:     container.ProfileStatService,
	})

	handlers.RegisterBlockedUserHandler(api, handlers.BlockedDeps{
		Service: container.BlockedUserService,
		Logger:  container.Logger,
	})

	health := http.HealthHandler("social-service", container.DB)
	g.GET("/social/health", health)
}
