package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(g *gin.Engine, di *app.Container) {
	api := g.Group("/api/v2")

	handlers.NewBugHandler(api, handlers.BugHandlerDeps{
		Logger:  di.Logger,
		UseCase: di.BugCases,
	})
}
