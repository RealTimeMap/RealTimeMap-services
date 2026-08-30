package handlers

import (
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/key"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ApiKeyDeps struct {
	Service *key.ApiKeyService

	Logger *zap.Logger
}

type ApiKeyHandler struct {
	service *key.ApiKeyService
	logger  *zap.Logger
}

func NewApiKeyHandler(g *gin.RouterGroup, deps ApiKeyDeps) {
	h := &ApiKeyHandler{
		service: deps.Service,
		logger:  deps.Logger,
	}

	r := g.Group("/keys")
	{
		r.POST("/create", auth.AdminOnly(), h.Create)
	}
}

func (h *ApiKeyHandler) Create(c *gin.Context) {
	var req ApiKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	token, err := h.service.CreateToken(c.Request.Context(), key.CreateKeyParams{
		Name:      req.Name,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, ApiKeyResponse{ApiKey: token})
}
