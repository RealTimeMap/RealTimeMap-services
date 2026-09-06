package handlers

import (
	"context"
	"net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/subscription"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SubscriptionDeps struct {
	Service *subscription.Service

	Logger *zap.Logger
}

type SubscriptionHandler struct {
	service *subscription.Service
	logger  *zap.Logger
}

func RegisterSubscriptionHandler(g *gin.RouterGroup, deps SubscriptionDeps) {
	h := &SubscriptionHandler{
		service: deps.Service,
		logger:  deps.Logger,
	}

	group := g.Group("/subscriptions")
	{
		group.POST("/subscribe", auth.AuthRequired(), h.SubscribeHandle)
		group.POST("/unsubscribe", auth.AuthRequired(), h.UnsubscribeHandle)

		// Исходящие подписки текущего пользователя
		group.GET("", auth.AuthRequired(), h.GetMySubscriptionsHandle)
		group.GET("/", auth.AuthRequired(), h.GetMySubscriptionsHandle)

		// Подписчики текущего пользователя
		group.GET("/subscribers", auth.AuthRequired(), h.GetMySubscribersHandle)
		group.GET("/subscribers/", auth.AuthRequired(), h.GetMySubscribersHandle)

		// Состояние подписки текущего пользователя на чужой профиль
		group.GET("/status/:profileID", auth.AuthRequired(), h.GetStatusHandle)
	}
}

func (h *SubscriptionHandler) SubscribeHandle(c *gin.Context) {
	subscriberID, targetID, ok := h.bindMutation(c)
	if !ok {
		return
	}
	if err := h.service.Subscribe(c.Request.Context(), subscriberID, targetID); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SubscriptionHandler) UnsubscribeHandle(c *gin.Context) {
	subscriberID, targetID, ok := h.bindMutation(c)
	if !ok {
		return
	}
	if err := h.service.Unsubscribe(c.Request.Context(), subscriberID, targetID); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SubscriptionHandler) GetMySubscriptionsHandle(c *gin.Context) {
	h.listProfiles(c, h.service.GetSubscriptionsProfile)
}

func (h *SubscriptionHandler) GetMySubscribersHandle(c *gin.Context) {
	h.listProfiles(c, h.service.GetSubscribersProfile)
}

func (h *SubscriptionHandler) GetStatusHandle(c *gin.Context) {
	uData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	targetID, err := middleware.ParsePathParams(c, "profileID")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	subscribed, err := h.service.IsSubscribed(c.Request.Context(), uint(uData.UserID), targetID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewSubscriptionStatusResponse(subscribed))
}

// bindMutation достаёт id текущего пользователя и id цели из тела запроса.
func (h *SubscriptionHandler) bindMutation(c *gin.Context) (uint, uint, bool) {
	uData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return 0, 0, false
	}

	var req dto.SubscriptionRequest
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return 0, 0, false
	}

	return uint(uData.UserID), req.UserID, true
}

// listProfiles общая обвязка для постраничных списков профилей.
func (h *SubscriptionHandler) listProfiles(
	c *gin.Context,
	load func(ctx context.Context, userID uint, params *subscription.SubscriptionSearchParams) ([]*model.Profile, int64, error),
) {
	uData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	var req dto.SubscriptionSearchParams
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	params := &subscription.SubscriptionSearchParams{Pagination: pagination.Params{
		Page:     req.Page,
		PageSize: req.PageSize,
	}}

	profiles, count, err := load(c.Request.Context(), uint(uData.UserID), params)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	items := dto.NewSearchProfileItems(profiles)
	c.JSON(http.StatusOK, pagination.NewResponse(items, params.Pagination, count))
}
