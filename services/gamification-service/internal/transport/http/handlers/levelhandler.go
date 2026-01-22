package handlers

import (
	"net/http"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/cache"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/app/dto"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelservice"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *levelservice.LevelService

	logger *zap.Logger
}
type LevelDeps struct {
	Service *levelservice.LevelService
	Cache   cache.Cache
	Logger  *zap.Logger
}

func RegisterLevelHandler(g *gin.RouterGroup, deps LevelDeps) {
	h := &Handler{service: deps.Service, logger: deps.Logger}
	r := g.Group("/level")
	{
		r.GET("/", cache.Middleware(deps.Cache, cache.Options{Prefix: "levels", TTL: 15 * time.Minute}), h.GetLevels)
	}
}

func (h *Handler) GetLevels(c *gin.Context) {

	levels, err := h.service.GetLevels(c.Request.Context())
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewMultiResponse(levels))
}
