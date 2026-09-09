package http

import (
	"github.com/gin-gonic/gin"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/transport/http/handlers"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/transport/http/middleware"
)

func RegisterRoutes(g *gin.Engine, di *app.Container) {
	api := g.Group("/api/v2", auth.NotBanned())

	handlers.NewBugHandler(api, handlers.BugHandlerDeps{
		Logger:  di.Logger,
		UseCase: di.BugCases,
	})

	// Межсервисные маршруты: сюда ходит таск-менеджер за перечнем багов
	// и с обратной синхронизацией статуса. Аутентификация по ключу
	// сервиса, а не по заголовкам пользователя: у фонового вызова
	// пользователя нет.
	service := g.Group("/api/v2/service", middleware.ServiceOnly(di.ServiceApiKey))
	handlers.NewServiceBugHandler(service, handlers.ServiceBugHandlerDeps{
		Logger:  di.Logger,
		UseCase: di.BugCases,
	})

	health := http.HealthHandler("feedback-service", di.DB)
	g.GET("/feedback/health", health)
}
