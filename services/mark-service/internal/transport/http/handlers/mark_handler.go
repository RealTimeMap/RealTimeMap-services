package handlers

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/valueobject"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/mark"
	"github.com/gin-gonic/gin"
	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"go.uber.org/zap"
)

type MarkHandler struct {
	service *service.MarkService
	logger  *zap.Logger
}

func InitMarkHandler(g *gin.RouterGroup, service *service.MarkService, logger *zap.Logger) {
	handler := &MarkHandler{service: service, logger: logger}
	markGroup := g.Group("/mark")
	{
		markGroup.POST("/create", auth.AuthRequired(), handler.CreateMark)
		markGroup.POST("/", handler.GetMarks)
	}
}

func (h *MarkHandler) CreateMark(c *gin.Context) {
	var request dto.RequestMark
	request.StartAt = time.Now()

	userID, userName, err := context.GetUserInfo(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "unauthorized"})
		return
	}

	if err := c.ShouldBind(&request); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	// Валидация media файлов
	photos := make([]mediavalidator.PhotoInput, 0, len(request.Photos))

	for i, fileHeader := range request.Photos {
		// 3.1. Проверка размера
		if fileHeader.Size > maxFileSize {
			validation.Abort(c, validation.NewFieldError(
				fmt.Sprintf("photos[%d]", i),
				"file size exceeds maximum allowed size of 5MB",
				"value_error.file.too_large",
				fileHeader.Size,
			))
			return
		}

		// 3.2. Проверка Content-Type header
		contentType := fileHeader.Header.Get("Content-Type")
		isAllowed := false
		for _, t := range allowedTypes {
			if t == contentType {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			validation.Abort(c, validation.NewFieldError(
				fmt.Sprintf("photos[%d]", i),
				"file type not allowed. Allowed types: jpeg, png, webp, svg",
				"value_error.mime_type",
				contentType,
			))
			return
		}

		// 3.3. Чтение файла
		file, err := fileHeader.Open()
		if err != nil {
			validation.Abort(c, validation.NewFieldError(
				fmt.Sprintf("photos[%d]", i),
				"failed to open uploaded file",
				"value_error.file.invalid",
				nil,
			))
			return
		}
		defer file.Close()

		photoData, err := io.ReadAll(file)
		if err != nil {
			validation.Abort(c, validation.NewFieldError(
				fmt.Sprintf("photos[%d]", i),
				"failed to read uploaded file",
				"value_error.file.invalid",
				nil,
			))
			return
		}

		// 3.4. Добавление в slice с чистыми данными
		photos = append(photos, mediavalidator.PhotoInput{
			Data:     photoData,
			FileName: fileHeader.Filename,
		})
	}

	// ✅ Конвертируем DTO → Value Object
	markName, err := valueobject.NewMarkName(request.MarkName)
	if err != nil {
		// Ошибка валидации имени метки (пустое, слишком короткое/длинное)
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	duration, err := valueobject.NewDuration(request.Duration)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	// Мапинг и создание
	validData := service.MarkInput{
		MarkName:       markName,
		AdditionalInfo: request.AdditionalInfo,
		Geom:           types.Point{Point: orb.Point{request.Longitude, request.Latitude}},
		Geohash:        geohash.EncodeWithPrecision(request.Latitude, request.Longitude, 5),
		CategoryId:     request.CategoryId,
		StartAt:        request.StartAt,
		Duration:       duration,
		Photos:         photos,
		UserID:         userID,
		UserName:       userName,
	}
	res, err := h.service.CreateMark(c.Request.Context(), validData)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(201, dto.NewResponseMark(res))
}

func (h *MarkHandler) GetMarks(c *gin.Context) {
	var params dto.FilterParams
	params.EndAt = time.Now().UTC()

	if err := c.ShouldBindBodyWithJSON(&params); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	validParams := repository.Filter{BoundingBox: valueobject.BoundingBox{
		LeftTop:     valueobject.Point{Lon: params.Screen.LeftTop.Longitude, Lat: params.Screen.LeftTop.Latitude},
		RightBottom: valueobject.Point{Lon: params.Screen.RightBottom.Longitude, Lat: params.Screen.RightBottom.Latitude},
	},
		StartAt: params.StartAt,
		EndAt:   params.EndAt,
	}
	marks, err := h.service.GetMarsInArea(c.Request.Context(), validParams)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(200, dto.NewMultiplyResponseMark(marks))
}
