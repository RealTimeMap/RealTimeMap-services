package handlers

import (
	"net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/blockeduser"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BlockedDeps struct {
	Service *blockeduser.Service

	Logger *zap.Logger
}

type BlockedUserHandler struct {
	service *blockeduser.Service
	logger  *zap.Logger
}

func RegisterBlockedUserHandler(g *gin.RouterGroup, deps BlockedDeps) {
	h := &BlockedUserHandler{
		service: deps.Service,
		logger:  deps.Logger,
	}
	blockedGroup := g.Group("/friends")
	{
		blockedGroup.POST("/block", auth.AuthRequired(), h.BlockUserHandle)
		blockedGroup.POST("/block/", auth.AuthRequired(), h.BlockUserHandle)

		blockedGroup.POST("/unblock", auth.AuthRequired(), h.UnBlockUserHandle)
		blockedGroup.POST("/unblock/", auth.AuthRequired(), h.UnBlockUserHandle)

		blockedGroup.GET("/blocked", auth.AuthRequired(), h.GetBlockedUsersHandle)
		blockedGroup.GET("/blocked/", auth.AuthRequired(), h.GetBlockedUsersHandle)
	}
}

func (h *BlockedUserHandler) BlockUserHandle(c *gin.Context) {
	uData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	var req dto.BlockedUserRequest
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	err = h.service.BlockUser(c.Request.Context(), uint(uData.UserID), req.UserID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *BlockedUserHandler) UnBlockUserHandle(c *gin.Context) {
	uData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	var req dto.BlockedUserRequest
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	err = h.service.UnBlockUser(c.Request.Context(), uint(uData.UserID), req.UserID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *BlockedUserHandler) GetBlockedUsersHandle(c *gin.Context) {
	uData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	var req dto.BlockedSearchParams
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	params := &blockeduser.BlockedSearchParams{Pagination: pagination.Params{
		Page:     req.Page,
		PageSize: req.PageSize,
	}}

	profiles, count, err := h.service.GetBlockedUsersProfile(c.Request.Context(), uint(uData.UserID), params)

	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	items := dto.NewSearchProfileItems(profiles)
	c.JSON(http.StatusOK, pagination.NewResponse(items, params.Pagination, count))

}
