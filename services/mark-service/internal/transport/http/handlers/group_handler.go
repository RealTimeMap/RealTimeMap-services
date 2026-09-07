package handlers

import (
	"net/http"
	"time"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	httputils "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/group"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GroupDeps struct {
	UseCases *group.Application
	Logger   *zap.Logger
}

type groupHandler struct {
	useCases *group.Application
	logger   *zap.Logger
}

func InitGroupHandler(g *gin.RouterGroup, deps GroupDeps) {
	h := &groupHandler{
		useCases: deps.UseCases,
		logger:   deps.Logger,
	}
	r := g.Group("/group")
	{
		r.POST("/create", auth.AuthRequired(), h.CreateGroup)
		r.GET("/list", auth.AuthRequired(), h.List)
	}
}

type (
	CreateGroupRequest struct {
		Name        string  `json:"name" binding:"required"`
		Description *string `json:"description"`
	}
	GroupResponse struct {
		ID          uint      `json:"id"`
		UserID      uint      `json:"userId"`
		Name        string    `json:"name"`
		Description *string   `json:"description"`
		CreatedAt   time.Time `json:"createdAt"`
	}
)

func ToGroupResponse(obj group.GroupResult) GroupResponse {
	return GroupResponse{
		ID:          obj.ID,
		UserID:      obj.UserID,
		Name:        obj.Name,
		Description: obj.Description,
		CreatedAt:   obj.CreatedAt,
	}
}

func (h *groupHandler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest

	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	obj, err := h.useCases.Create.Handle(c.Request.Context(), group.CreateGroupCommand{
		Name:        req.Name,
		Description: req.Description,
		UserID:      uint(userInfo.UserID),
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, ToGroupResponse(obj))
}

func (h *groupHandler) List(c *gin.Context) {
	var req pagination.Params

	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	req.Defaults()
	objs, count, err := h.useCases.List.Handle(c.Request.Context(), group.GetListCommand{
		Params: req,
		UserID: uint(userInfo.UserID),
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	res := httputils.Map(objs, ToGroupResponse)

	c.JSON(http.StatusOK, pagination.NewResponse(res, req, count))
}
