package handlers

import (
	"net/http"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_stat"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type statHandler struct {
	useCases *comment_stat.Application
	logger   *zap.Logger
}

type StatDeps struct {
	UseCases *comment_stat.Application
	Logger   *zap.Logger
}

func InitStatHandler(g *gin.RouterGroup, deps StatDeps) {
	h := &statHandler{useCases: deps.UseCases, logger: deps.Logger}
	statGroup := g.Group("/stats")
	{
		statGroup.GET("/", h.Stat)
	}
}

func (h *statHandler) Stat(c *gin.Context) {
	var req date.Query

	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	period, err := req.Resolve(time.Now(), time.UTC)
	if err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, err := h.useCases.GetStat.Handle(c.Request.Context(), comment_stat.GetStatCommand{
		UserID: 1,
		Period: period,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"stat1": res.Current,
		"stat2": res.Previous,
	})
}
