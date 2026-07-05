package handlers

import (
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/category"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/category"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var allowedTypes = [4]string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"}

type CategoryHandler struct {
	useCase *category.Application
	logger  *zap.Logger
}

func InitCategoryHandler(g *gin.RouterGroup, useCase *category.Application, logger *zap.Logger) {
	handler := &CategoryHandler{
		useCase: useCase,
		logger:  logger,
	}

	categoryGroup := g.Group("/category")
	{
		// Support both with and without trailing slash
		categoryGroup.POST("", auth.AdminOnly(), handler.CreateCategory)
		categoryGroup.POST("/", auth.AdminOnly(), handler.CreateCategory)
		categoryGroup.GET("/all", handler.GetCategories)
	}
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req dto.RequestCategory

	// 1. Валидация структуры запроса (binding tags)
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	// 4. Вызов сервиса с чистыми данными
	newCategory, err := h.useCase.Create.Handle(c.Request.Context(), category.CreateCategoryCommand{
		Icon:         req.Icon,
		Color:        req.Color,
		CategoryName: req.CategoryName,
	})
	if err != nil {
		// 5. Обработка ошибок от сервисного слоя
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	h.logger.Info("create category success", zap.Int("categoryId", int(newCategory.ID)))
	// 6. Успешный ответ
	c.JSON(http.StatusCreated, dto.NewResponseCategory(newCategory))
}

func (h *CategoryHandler) GetCategories(c *gin.Context) {
	res, err := h.useCase.Get.Handle(c.Request.Context())
	if err != nil {
		h.logger.Error("get categories", zap.Error(err))
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewMultiResponseCategory(res))
}
