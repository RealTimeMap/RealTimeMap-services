package handler

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service/stats"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type StatHandler struct {
	service *stats.CommentStatsService
	logger  *zap.Logger
}

func RegisterStatHandler(g *gin.RouterGroup, service *stats.CommentStatsService, logger *zap.Logger) {
	h := &StatHandler{
		service: service,
		logger:  logger,
	}

	statGroup := g.Group("/stats")
	{
		statGroup.GET("/", h.Stat)
	}
}

func (h *StatHandler) Stat(c *gin.Context) {
	var req date.Query

	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	period, err := req.Resolve(time.Now(), time.UTC)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c1, c2, err := h.service.GetStat(c.Request.Context(), 1, period)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(200, gin.H{
		"stat1": c1,
		"stat2": c2,
	})
}
