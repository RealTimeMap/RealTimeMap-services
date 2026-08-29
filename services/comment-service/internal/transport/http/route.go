package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2")

	handlers.InitCommentHandler(api, handlers.CommentDeps{UseCases: container.CommentUseCases, Logger: container.Logger})
	handlers.InitReactionHandler(api, handlers.ReactionDeps{UseCases: container.InteractionUseCases, Logger: container.Logger})
	handlers.InitStatHandler(api, handlers.StatDeps{UseCases: container.StatUseCases, Logger: container.Logger})

	health := http.HealthHandler("comment-service", container.DB)
	g.GET("/comment/health", health)
}
