package handlers

import (
	"net/http"
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type StatHandler struct {
	service *profile.StatService
	logger  *zap.Logger
}

type StatDeps struct {
	Service     *profile.StatService
	ProfileRepo repository.ProfileRepository
	Logger      *zap.Logger
}

func RegisterStatHandler(g *gin.RouterGroup, deps StatDeps) {
	handler := &StatHandler{
		service: deps.Service,
		logger:  deps.Logger,
	}
	statGroup := g.Group("/:profileID")
	{
		statGroup.GET("/statistics/summary", middleware.Exist(deps.ProfileRepo.Exist, handler.logger, "profileID"), handler.withProfileID(handler.GetProfileSummaryStat))
		statGroup.GET("/statistics/monthly", middleware.Exist(deps.ProfileRepo.Exist, handler.logger, "profileID"), handler.withProfileID(handler.GetProfileMonthlyActivity))
		statGroup.GET("/statistics/heatmap", middleware.Exist(deps.ProfileRepo.Exist, handler.logger, "profileID"), handler.withProfileID(handler.GetHeatMap))
		statGroup.GET("/statistics/categories", middleware.Exist(deps.ProfileRepo.Exist, handler.logger, "profileID"), handler.withProfileID(handler.GetPopularUserCategories))
	}
}

//func (h *StatHandler) GetStatBlock(c *gin.Context) {
//	var req date.Query
//	if err := c.ShouldBind(&req); err != nil {
//		errorhandler.HandleError(c, err, h.logger)
//		return
//	}
//
//	resolve, err := req.Resolve(time.Now(), time.UTC)
//	if err != nil {
//		errorhandler.HandleError(c, err, h.logger)
//		return
//	}
//
//	cS, cE := resolve.Current()
//	pS, pE := resolve.Previous()
//	c.JSON(200, gin.H{
//		"status":        "ok",
//		"currentStart":  cS,
//		"currentEnd":    cE,
//		"previousStart": pS,
//		"previousEnd":   pE,
//	})
//
//}

func (h *StatHandler) GetProfileSummaryStat(c *gin.Context, pID uint) {
	marks, friends, subs, err := h.service.GetProfileSummaryStat(c.Request.Context(), pID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	res := dto.NewSummaryProfileStat(marks, friends, subs)
	c.JSON(http.StatusOK, res)
}

func (h *StatHandler) GetProfileMonthlyActivity(c *gin.Context, pID uint) {
	res, err := h.service.GetUserMonthlyActivity(c.Request.Context(), pID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewMultipleMonthlyActivity(res))
}

func (h *StatHandler) GetHeatMap(c *gin.Context, pID uint) {
	var req dto.DateRangeParam
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	req.Defaults()
	activities, err := h.service.GetUserMarksHeatMap(c.Request.Context(), pID, req.Start, req.End)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewHeatmapResponse(activities, req.Start, req.End))

}

func (h *StatHandler) GetPopularUserCategories(c *gin.Context, pID uint) {
	result, err := h.service.GetPopularCategories(c.Request.Context(), pID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	response := dto.NewMultiplePopularCategoryResponse(result)
	c.JSON(http.StatusOK, response)
}

func parseProfileID(c *gin.Context) (uint, error) {
	pID, err := strconv.Atoi(c.Param("profileID"))
	if err != nil {
		err = apperror.NewFieldValidationError("profileID", "profileID must be a number", "value_error", c.Param("profileID"))
		return 0, err
	}
	return uint(pID), nil
}

func (h *StatHandler) withProfileID(fn func(*gin.Context, uint)) gin.HandlerFunc {
	return func(c *gin.Context) {
		pID, err := parseProfileID(c)
		if err != nil {
			errorhandler.HandleError(c, err, h.logger)
			return
		}
		fn(c, pID)
	}
}
