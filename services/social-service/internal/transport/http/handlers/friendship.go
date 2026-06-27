package handlers

import (
	"net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/friendship"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FriendshipDeps struct {
	Service *friendship.Service

	Logger *zap.Logger
}

type FriendshipHandler struct {
	service *friendship.Service
	logger  *zap.Logger
}

func RegisterFriendshipHandler(g *gin.RouterGroup, deps FriendshipDeps) {
	h := &FriendshipHandler{
		service: deps.Service,
		logger:  deps.Logger,
	}
	group := g.Group("/friends")
	{
		group.POST("/request", auth.AuthRequired(), h.SendRequestHandle)
		group.POST("/accept", auth.AuthRequired(), h.AcceptRequestHandle)
		group.POST("/decline", auth.AuthRequired(), h.DeclineRequestHandle)
		group.POST("/remove", auth.AuthRequired(), h.RemoveHandle)
		group.GET("", auth.AuthRequired(), h.GetFriendsHandle)
		group.GET("/", auth.AuthRequired(), h.GetFriendsHandle)
	}
}

func (h *FriendshipHandler) SendRequestHandle(c *gin.Context) {
	userID, friendID, ok := h.bindMutation(c)
	if !ok {
		return
	}
	if err := h.service.SendRequest(c.Request.Context(), userID, friendID); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *FriendshipHandler) AcceptRequestHandle(c *gin.Context) {
	userID, friendID, ok := h.bindMutation(c)
	if !ok {
		return
	}
	if err := h.service.AcceptRequest(c.Request.Context(), userID, friendID); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *FriendshipHandler) DeclineRequestHandle(c *gin.Context) {
	userID, friendID, ok := h.bindMutation(c)
	if !ok {
		return
	}
	if err := h.service.DeclineRequest(c.Request.Context(), userID, friendID); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *FriendshipHandler) RemoveHandle(c *gin.Context) {
	userID, friendID, ok := h.bindMutation(c)
	if !ok {
		return
	}
	if err := h.service.Remove(c.Request.Context(), userID, friendID); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *FriendshipHandler) GetFriendsHandle(c *gin.Context) {
	uData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	var req dto.FriendsSearchParams
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	params := &friendship.FriendsSearchParams{Pagination: pagination.Params{
		Page:     req.Page,
		PageSize: req.PageSize,
	}}

	profiles, count, err := h.service.GetFriendsProfile(c.Request.Context(), uint(uData.UserID), params)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	items := dto.NewSearchProfileItems(profiles)
	c.JSON(http.StatusOK, pagination.NewResponse(items, params.Pagination, count))
}

func (h *FriendshipHandler) bindMutation(c *gin.Context) (userID, friendID uint, ok bool) {
	uData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return 0, 0, false
	}
	var req dto.FriendRequest
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return 0, 0, false
	}
	return uint(uData.UserID), req.UserID, true
}
