package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2")
	handler.NewCommentRoute(api, container.CommentService, container.Logger)

}
