package handlers

import (
	"strconv"
	"time"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service/input"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/valueobject"
	subdtocat "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/dto/category"
	subdto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/dto/mark"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/mark"
	"github.com/gin-gonic/gin"
	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"go.uber.org/zap"
)

type MarkHandler struct {
	service *service.UserMarkService
	logger  *zap.Logger
}

func InitMarkHandler(g *gin.RouterGroup, service *service.UserMarkService, logger *zap.Logger) {
	handler := &MarkHandler{service: service, logger: logger}
	markGroup := g.Group("/marks")
	{
		markGroup.POST("/", handler.GetMarks)
		markGroup.GET("/my", auth.AuthRequired(), handler.GetUserMarks)
		markGroup.GET("/create-data", handler.GetDataForCreate)
		markGroup.POST("/create", auth.AuthRequired(), handler.CreateMark)
		markGroup.GET("/:markID", handler.DetailMark)
		markGroup.DELETE("/:markID", auth.AuthRequired(), handler.DeleteMark)
		markGroup.PATCH("/:markID", auth.AuthRequired(), handler.UpdateMark)
	}
}

func (h *MarkHandler) CreateMark(c *gin.Context) {
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

	markName, err := valueobject.NewMarkName(request.MarkName)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	// Маппинг в чистые данные для Service Layer (Clean Architecture)
	validData := input.MarkInput{
		MarkName:       markName,
		AdditionalInfo: request.AdditionalInfo,
		Geom:           types.Point{Point: orb.Point{request.Longitude, request.Latitude}},
		Geohash:        geohash.EncodeWithPrecision(request.Latitude, request.Longitude, 5),
		CategoryId:     request.CategoryId,
		StartAt:        request.StartAt,
		EndAt:          request.EndAt,
		Photos:         photos, // Чистые данные []PhotoInput
		UserInput:      userInfo,
	}
	res, err := h.service.CreateMark(c.Request.Context(), validData)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(201, dto.NewResponseMark(res))
}

func (h *MarkHandler) GetMarks(c *gin.Context) {
	var params subdto.FilterParams
	params.ZoomLevel = 15
	const zoomSelector = 12
	params.EndAt = time.Now().UTC()

	if err := c.ShouldBindBodyWithJSON(&params); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	validParams := subdto.ToInputFilter(params)
	if validParams.ZoomLevel < zoomSelector {
		clusters, err := h.service.GetMarksInCluster(c.Request.Context(), validParams)
		if err != nil {
			errorhandler.HandleError(c, err, h.logger)
			return
		}
		c.JSON(200, dto.NewMultipleResponseCluster(clusters))
	} else {
		marks, err := h.service.GetMarksInArea(c.Request.Context(), validParams)
		if err != nil {
			errorhandler.HandleError(c, err, h.logger)
			return
		}

		c.JSON(200, dto.NewMultipleResponseMark(marks))
	}
}

func (h *MarkHandler) DeleteMark(c *gin.Context) {
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	markID, err := strconv.Atoi(c.Param("markID"))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	err = h.service.DeleteMark(c.Request.Context(), markID, userInfo)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(204)
}

func (h *MarkHandler) UpdateMark(c *gin.Context) {
	var req dto.RequestUpdateMark

	markID, err := strconv.Atoi(c.Param("markID"))
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

	validData := input.MarkUpdateInput{
		MarkID:         markID,
		Photos:         photos, // Чистые данные []PhotoInput
		CategoryId:     req.CategoryId,
		AdditionalInfo: req.AdditionalInfo,
		UserInput:      userInfo,
		PhotosToDelete: req.PhotosToDelete,
	}
	if req.MarkName != nil {
		markName, err := valueobject.NewMarkName(*req.MarkName)
		if err != nil {
			validation.AbortWithBindingError(c, err)
			return
		}
		validData.MarkName = &markName
	}
	if req.Duration != nil {
		duration, err := valueobject.NewDuration(*req.Duration)
		if err != nil {
			validation.AbortWithBindingError(c, err)
			return
		}
		validData.Duration = &duration
	}

	updatedMark, err := h.service.UpdateMark(c.Request.Context(), validData)

	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(200, dto.NewResponseMark(updatedMark))
}

func (h *MarkHandler) DetailMark(c *gin.Context) {
	markID, err := strconv.Atoi(c.Param("markID"))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	mark, err := h.service.DetailMark(c.Request.Context(), markID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(200, dto.NewResponseMark(mark))
}

func (h *MarkHandler) GetDataForCreate(c *gin.Context) {

	categories, durations, err := h.service.GetDataForCreate(c.Request.Context())
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	response := subdtocat.NewResponseCreateData(categories, durations)
	c.JSON(200, response)
}

func (h *MarkHandler) GetUserMarks(c *gin.Context) {
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	var params pagination.Params
	if err := c.ShouldBindQuery(&params); err != nil {
		validation.AbortWithBindingError(c, err)
	}

	marks, count, err := h.service.GetUserMarks(c.Request.Context(), uint(userInfo.UserID), params)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	response := dto.NewMultipleResponseMark(marks)
	c.JSON(200, pagination.NewResponse(response, params, count))
}
