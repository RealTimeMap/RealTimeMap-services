package http

import (
	"github.com/gin-gonic/gin"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http/handlers"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2", auth.NotBanned())

	handlers.InitCommentHandler(api, handlers.CommentDeps{UseCases: container.CommentUseCases, Logger: container.Logger})
	handlers.InitReactionHandler(api, handlers.ReactionDeps{UseCases: container.InteractionUseCases, Logger: container.Logger})
	handlers.InitStatHandler(api, handlers.StatDeps{UseCases: container.StatUseCases, Logger: container.Logger})

	health := http.HealthHandler("comment-service", container.DB)
	g.GET("/comment/health", health)
}
