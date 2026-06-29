package handlers

import (
	"net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service/accrual"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AccrualDeps struct {
	Service *accrual.Service

	Logger *zap.Logger
}

type AccrualHandler struct {
	service *accrual.Service
	logger  *zap.Logger
}

func RegisterAccrualHandler(g *gin.RouterGroup, deps AccrualDeps) {
	h := &AccrualHandler{service: deps.Service, logger: deps.Logger}

	accrualGroup := g.Group("/marks/:markID")
	{
		accrualGroup.POST("/share", h.ShareHandle)
		accrualGroup.POST("/like", auth.AuthRequired(), h.LikeHandle)
	}
}

func (h *AccrualHandler) ShareHandle(c *gin.Context) {
	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	count, err := h.service.IncreaseShare(c.Request.Context(), markID)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

func (h *AccrualHandler) LikeHandle(c *gin.Context) {
	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	userID, err := helper.GetUserID(c)

	err = h.service.SetLike(c.Request.Context(), markID, uint(userID))
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
