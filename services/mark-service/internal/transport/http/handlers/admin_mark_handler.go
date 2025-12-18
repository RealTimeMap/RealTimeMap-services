package handlers

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/mark"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AdminMarkHandler struct {
	service *service.AdminMarkService
	logger  *zap.Logger
}

func InitAdminMarkHandler(g *gin.RouterGroup, service *service.AdminMarkService, logger *zap.Logger) {
	handler := &AdminMarkHandler{service: service, logger: logger}
	group := g.Group("/admin/mark")
	{
		group.GET("/", auth.AdminOnly(), handler.GetAll)
	}
}

func (h *AdminMarkHandler) GetAll(c *gin.Context) {
	var params pagination.Params
	params.Defaults()
	if err := c.ShouldBindQuery(&params); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	marks, count, err := h.service.GetAll(c.Request.Context(), params)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	marksResponse := dto.NewMultipleResponseMark(marks)
	response := pagination.NewResponse(marksResponse, params, count)
	c.JSON(200, response)
}
