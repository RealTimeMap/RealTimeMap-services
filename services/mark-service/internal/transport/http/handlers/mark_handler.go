package handlers

import (
	"fmt"
	"io"
	"time"

	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
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
		markGroup.POST("/", handler.CreateMark)
	}
}

func (h *MarkHandler) CreateMark(c *gin.Context) {
	var request dto.RequestMark
	request.StartAt = time.Now()
	if err := c.ShouldBind(&request); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	// Валидация media файлов
	photos := make([]service.PhotoInput, 0, len(request.Photos))

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
		photos = append(photos, service.PhotoInput{
			Data:     photoData,
			FileName: fileHeader.Filename,
		})
	}

	// Мапинг и создание
	validData := service.MarkInput{
		MarkName:       request.MarkName,
		AdditionalInfo: request.AdditionalInfo,
		Geom:           types.Point{Point: orb.Point{request.Longitude, request.Latitude}},
		Geohash:        geohash.EncodeWithPrecision(request.Latitude, request.Longitude, 5),
		CategoryId:     request.CategoryId,
		StartAt:        request.StartAt,
		Duration:       request.Duration,
		Photos:         photos,
	}
	res, err := h.service.CreateMark(c.Request.Context(), validData)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(201, dto.NewResponseMark(res))
}
