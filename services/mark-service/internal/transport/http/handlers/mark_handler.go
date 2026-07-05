package handlers

import (
	"time"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_action"
	subdto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/dto/mark"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/mark"
	"github.com/gin-gonic/gin"
	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"go.uber.org/zap"
)

type markHandler struct {
	useCases *mark_action.Application
	logger   *zap.Logger
}

type MarkDeps struct {
	UseCases *mark_action.Application
	Logger   *zap.Logger
}

func InitMarkHandler(g *gin.RouterGroup, deps MarkDeps) {
	handler := &markHandler{logger: deps.Logger, useCases: deps.UseCases}
	markGroup := g.Group("/marks")
	{
		markGroup.POST("/", handler.GetMarks)
		markGroup.GET("/:markID/list", handler.GetUserMarks) // markID потому что особенность путей, подразумевается userID
		markGroup.POST("/create", auth.AuthRequired(), handler.CreateMark)
		markGroup.GET("/:markID", handler.DetailMark)
		markGroup.DELETE("/:markID", auth.AuthRequired(), handler.DeleteMark)
		markGroup.PATCH("/:markID", auth.AuthRequired(), handler.UpdateMark)
	}
}

func (h *markHandler) CreateMark(c *gin.Context) {
	var request dto.RequestMark

	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err := c.ShouldBind(&request); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	// Чтение и валидация фотографий (параллельно, с проверкой MIME из байтов)
	photos, err := processPhotoUploads(request.Photos)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	// Маппинг в чистые данные для Service Layer (Clean Architecture)
	cmd := mark_action.MarkCreateCommand{
		MarkName:       request.MarkName,
		AdditionalInfo: request.AdditionalInfo,
		Geom:           types.Point{Point: orb.Point{request.Longitude, request.Latitude}},
		Geohash:        geohash.EncodeWithPrecision(request.Latitude, request.Longitude, 5),
		CategoryId:     request.CategoryId,
		StartAt:        request.StartAt,
		EndAt:          request.EndAt,
		Photos:         photos,
		User:           userInfo,
	}

	res, err := h.useCases.CreateMark.Handle(c.Request.Context(), cmd)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(201, dto.NewResponseMarkV2(res))
}

func (h *markHandler) GetMarks(c *gin.Context) {
	var params subdto.FilterParams
	params.ZoomLevel = 15
	params.EndAt = time.Now().UTC()

	if err := c.ShouldBindBodyWithJSON(&params); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	validParams := subdto.ToInputFilterV2(params)
	res, err := h.useCases.GetMark.Handle(c.Request.Context(), validParams)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	switch res.Mode {
	case mark_action.ModeCluster:
		c.JSON(200, gin.H{"marks": res.Marks})
		return
	default:
		c.JSON(200, gin.H{"marks": res.Marks})
		return
	}
}

func (h *markHandler) DeleteMark(c *gin.Context) {
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	err = h.useCases.DeleteMark.Handle(c.Request.Context(), mark_action.DeleteMarkCommand{
		MarkID: markID,
		UserID: uint(userInfo.UserID),
	})

	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(204)
}

func (h *markHandler) UpdateMark(c *gin.Context) {
	var req dto.RequestUpdateMark

	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	// Чтение и валидация фотографий (параллельно, с проверкой MIME из байтов)
	photos, err := processPhotoUploads(req.Photos)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	var markName string
	if req.MarkName != nil {
		markName = *req.MarkName
	}

	updatedMark, err := h.useCases.UpdateMark.Handle(c.Request.Context(), mark_action.MarkUpdateCommand{
		MarkID:         markID,
		MarkName:       markName,
		UserID:         uint(userInfo.UserID),
		EndAt:          req.EndAt,
		AdditionalInfo: req.AdditionalInfo,

		PhotosToDelete: req.PhotosToDelete,
		Photos:         photos,
	})

	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(200, dto.NewResponseMarkV2(updatedMark))
}

func (h *markHandler) DetailMark(c *gin.Context) {
	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	mark, err := h.useCases.GetDetail.Handle(c.Request.Context(), markID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(200, dto.NewDetailMarkResponse(mark))
}

func (h *markHandler) GetUserMarks(c *gin.Context) {
	userID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	var params pagination.Params
	if err := c.ShouldBindQuery(&params); err != nil {
		validation.AbortWithBindingError(c, err)
	}

	marks, count, err := h.useCases.GetUserMark.Handle(c.Request.Context(), mark_action.UserMarkGetterCommand{
		UserID: userID,
		Params: params,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	response := dto.NewMultipleResponseMarkV2(marks)
	c.JSON(200, pagination.NewResponse(response, params, count))
}
