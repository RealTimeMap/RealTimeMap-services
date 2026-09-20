package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	ctxhelper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/transport/http/dto"
)

type tokenHandler struct {
	service *token.Service

	logger *zap.Logger
}

type TokenDeps struct {
	Service *token.Service

	Logger *zap.Logger
}

func InitTokenHandler(g *gin.RouterGroup, deps TokenDeps) {
	h := &tokenHandler{
		service: deps.Service,
		logger:  deps.Logger,
	}
	r := g.Group("/tokens")
	{
		r.POST("", auth.AuthRequired(), h.Create)
		r.PATCH("/me", auth.AuthRequired(), h.UpdateSettings)
	}
}

func (h *tokenHandler) Create(c *gin.Context) {
	userID, err := ctxhelper.GetUserID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	var req dto.CreateTokenRequest
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	if err := h.service.Register(c.Request.Context(), req.ToParams(uint(userID))); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.Status(http.StatusNoContent)
}

// UpdateSettings меняет настройки уведомлений устройства, с которого пришёл
func (h *tokenHandler) UpdateSettings(c *gin.Context) {
	userID, err := ctxhelper.GetUserID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	var req dto.UpdateSettingsRequest
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	device, err := h.service.UpdateSettings(c.Request.Context(), req.ToParams(uint(userID)))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewSettingsResponse(device))
}
