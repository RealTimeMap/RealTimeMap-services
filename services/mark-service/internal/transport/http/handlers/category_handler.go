package handlers

import (
	"io"
	"mime/multipart"
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/category"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var allowedTypes = [4]string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"}

type CategoryHandler struct {
	service *service.CategoryService
	logger  *zap.Logger
}

func InitCategoryHandler(g *gin.RouterGroup, service *service.CategoryService, logger *zap.Logger) {
	handler := &CategoryHandler{
		service: service,
		logger:  logger,
	}

	categoryGroup := g.Group("/category")
	{
		// Support both with and without trailing slash
		categoryGroup.POST("", auth.AdminOnly(), handler.CreateCategory)
		categoryGroup.POST("/", auth.AdminOnly(), handler.CreateCategory)
	}
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req dto.RequestCategory

	// 1. Валидация структуры запроса (binding tags)
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	// 2. HTTP-уровень валидации файла
	// Проверка размера файла
	if req.Icon.Size > maxFileSize {
		validation.Abort(c, validation.NewFieldError(
			"icon",
			"file size exceeds maximum allowed size of 5MB",
			"value_error.file.too_large",
			req.Icon.Size,
		))
		return
	}

	// Проверка Content-Type header (базовая проверка на HTTP уровне)
	contentType := req.Icon.Header.Get("Content-Type")

	isAllowed := false
	for _, t := range allowedTypes {
		if t == contentType {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		validation.Abort(c, validation.NewFieldError(
			"icon",
			"file type not allowed. Allowed types: jpeg, png, webp, svg",
			"value_error.mime_type",
			contentType,
		))
		return
	}

	// 3. Чтение файла
	file, err := req.Icon.Open()
	if err != nil {
		validation.Abort(c, validation.NewFieldError(
			"icon",
			"failed to open uploaded file",
			"value_error.file.invalid",
			nil,
		))
		return
	}
	defer file.Close()

	iconData, err := io.ReadAll(file)
	if err != nil {
		validation.Abort(c, validation.NewFieldError(
			"icon",
			"failed to read uploaded file",
			"value_error.file.invalid",
			nil,
		))
		return
	}

	// 4. Вызов сервиса с чистыми данными
	newCategory, err := h.service.CreateCategory(c.Request.Context(), service.CategoryCreateInput{
		CategoryName: req.CategoryName,
		Color:        req.Color,
		IconData:     iconData,
		FileName:     req.Icon.Filename,
	})
	if err != nil {
		// 5. Обработка ошибок от сервисного слоя
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	// 6. Успешный ответ
	c.JSON(http.StatusCreated, dto.NewResponseCategory(newCategory))
}

func (h *CategoryHandler) validateMedia(inputFile *multipart.FileHeader) error { return nil }
