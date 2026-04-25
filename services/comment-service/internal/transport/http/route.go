package http

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2")
	h := &Handler{
		DB: container.DB,
	}
	handler.NewCommentRoute(api, container.CommentService, container.Logger)
	api.GET("/comment/health", h.healthHandler)
}

func (h *Handler) healthHandler(c *gin.Context) {

	sqlDB, err := h.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		c.JSON(503, gin.H{
			"status":   "unhealthy",
			"database": "down",
		})
		return
	}

	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "comment-service",
	})
}
