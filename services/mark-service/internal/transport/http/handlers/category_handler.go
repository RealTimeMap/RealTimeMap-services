package handlers

import (
	"io"
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/category"
	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	service *service.CategoryService
}

func InitCategoryHandler(g *gin.RouterGroup, service *service.CategoryService) {
	handler := &CategoryHandler{service: service}
	categoryGroup := g.Group("/category")
	{
		categoryGroup.POST("/", auth.AdminOnly(), handler.CreateCategory)
	}
}

func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req dto.RequestCategory

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // TODO сделать обработчик ошибкок как в текущем проете
		return
	}

	file, err := req.Icon.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	defer file.Close()

	iconData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	newCategory, err := h.service.CreateCategory(c.Request.Context(), service.CategoryCreateInput{
		CategoryName: req.CategoryName,
		Color:        req.Color,
		IconData:     iconData,
		FileName:     req.Icon.Filename,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create category"}) // TODO обработка!
		return
	}
	c.JSON(http.StatusCreated, dto.NewResponseCategory(newCategory))
}
