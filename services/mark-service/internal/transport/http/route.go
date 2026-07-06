package http

import (
	"fmt"
	"io"
	"mime/multipart"
	"sync"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_action"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/handlers"
	"github.com/gin-gonic/gin"
	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"go.uber.org/zap"

	http2 "net/http"
)

func RegisterRoutes(g *gin.Engine, container *app.Container) {
	api := g.Group("/api/v2")

	// endpoints

	handlers.InitCategoryHandler(api, container.CategoryUseCases, container.Logger)
	handlers.InitMarkHandler(api, handlers.MarkDeps{UseCases: container.MarkUseCases, Logger: container.Logger})
	//handlers.RegisterAccrualHandler(api, handlers.AccrualDeps{Service: container.AccrualService, Logger: container.Logger})

	// Health
	health := http.HealthHandler("mark_action-service", container.DB)
	g.GET("/mark/health", health)

	h := H{
		create: container.MarkUseCases.CreateMark,
		logger: container.Logger,
	}
	api.POST("/create-test", auth.AuthRequired(), h.CreateV2)
	// SOCKET

	g.GET("/socket.io/*any", gin.WrapH(container.Socket.HttpHandler()))
	g.POST("/socket.io/*any", gin.WrapH(container.Socket.HttpHandler()))

}

type H struct {
	create *mark_action.CreateMarkHandler
	logger *zap.Logger
}

func (h *H) CreateV2(c *gin.Context) {
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	var req dto.RequestMark
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	photos, err := processPhotoUploads(req.Photos)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	// Маппинг в чистые данные для Service Layer (Clean Architecture)
	validData := mark_action.MarkCreateCommand{
		MarkName:       req.MarkName,
		AdditionalInfo: req.AdditionalInfo,
		Geom:           types.Point{Point: orb.Point{req.Longitude, req.Latitude}},
		Geohash:        geohash.EncodeWithPrecision(req.Latitude, req.Longitude, 5),
		CategoryId:     req.CategoryId,
		StartAt:        req.StartAt,
		EndAt:          req.EndAt,
		Photos:         photos,
		User:           userInfo,
	}

	mark, err := h.create.Handle(c.Request.Context(), validData)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	h.logger.Info("mark created", zap.Any("mark", mark))
	c.JSON(201, dto.NewResponseMark(mark))
}

const (
	maxFileSize      = 5 * 1024 * 1024 // 5 MB
	maxPhotosPerMark = 10
)

var allowedMimeTypes = []string{"image/jpeg", "image/png", "image/webp"}

func processPhotoUploads(fileHeaders []*multipart.FileHeader) ([]mediavalidator.PhotoInput, error) {
	if len(fileHeaders) == 0 {
		return nil, nil
	}

	// Проверка количества
	if len(fileHeaders) > maxPhotosPerMark {
		return nil, apperror.NewFieldValidationError(
			"photos",
			fmt.Sprintf("too many photos. Maximum allowed: %d, received: %d", maxPhotosPerMark, len(fileHeaders)),
			"value_error.list.max_items",
			len(fileHeaders),
		)
	}

	// Параллельное чтение файлов в память
	photos, err := readFilesParallel(fileHeaders)
	if err != nil {
		return nil, err
	}

	return photos, nil
}

// readFilesParallel читает файлы параллельно и валидирует MIME type из реальных байтов
func readFilesParallel(fileHeaders []*multipart.FileHeader) ([]mediavalidator.PhotoInput, error) {
	type readResult struct {
		photo mediavalidator.PhotoInput
		index int
		err   error
	}

	resultChan := make(chan readResult, len(fileHeaders))
	var wg sync.WaitGroup

	// Параллельное чтение
	for i, fh := range fileHeaders {
		wg.Add(1)
		go func(index int, header *multipart.FileHeader) {
			defer wg.Done()

			photo, err := readSingleFile(index, header)
			resultChan <- readResult{photo: photo, index: index, err: err}
		}(i, fh)
	}

	// Закрыть канал после завершения
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Сбор результатов с сохранением порядка
	results := make([]readResult, 0, len(fileHeaders))
	for result := range resultChan {
		if result.err != nil {
			return nil, result.err // Возвращаем первую ошибку
		}
		results = append(results, result)
	}

	// Сортировка по индексу для сохранения порядка
	photos := make([]mediavalidator.PhotoInput, len(fileHeaders))
	for _, r := range results {
		photos[r.index] = r.photo
	}

	return photos, nil
}

// readSingleFile читает один файл и валидирует его
func readSingleFile(index int, header *multipart.FileHeader) (mediavalidator.PhotoInput, error) {
	// Проверка размера
	if header.Size > maxFileSize {
		return mediavalidator.PhotoInput{}, apperror.NewFieldValidationError(
			fmt.Sprintf("photos[%d]", index),
			fmt.Sprintf("file size exceeds maximum allowed size of %d MB", maxFileSize/(1024*1024)),
			"value_error.file.too_large",
			header.Size,
		)
	}

	// Открыть файл
	file, err := header.Open()
	if err != nil {
		return mediavalidator.PhotoInput{}, apperror.NewFieldValidationError(
			fmt.Sprintf("photos[%d]", index),
			"failed to open uploaded file",
			"value_error.file.open",
			nil,
		)
	}
	defer file.Close()

	// Читаем в память
	data, err := io.ReadAll(file)
	if err != nil {
		return mediavalidator.PhotoInput{}, apperror.NewFieldValidationError(
			fmt.Sprintf("photos[%d]", index),
			"failed to read uploaded file",
			"value_error.file.read",
			nil,
		)
	}

	// Валидация MIME type из реальных байтов (не из HTTP заголовка!)
	mimeType := http2.DetectContentType(data)
	if !isAllowedMimeType(mimeType) {
		return mediavalidator.PhotoInput{}, apperror.NewInvalidMimeTypeError(
			fmt.Sprintf("photos[%d]", index),
			allowedMimeTypes,
			mimeType,
		)
	}

	return mediavalidator.PhotoInput{
		Data:     data,
		FileName: header.Filename,
	}, nil
}

// isAllowedMimeType проверяет допустимость MIME типа
func isAllowedMimeType(mimeType string) bool {
	for _, allowed := range allowedMimeTypes {
		if allowed == mimeType {
			return true
		}
	}
	return false
}
