package handlers

import (
	"net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app/use_cases/bug"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type BugHandlerDeps struct {
	UseCase *bug.Application

	Logger *zap.Logger
}

type BugHandler struct {
	useCase *bug.Application

	logger *zap.Logger
}

func NewBugHandler(g *gin.RouterGroup, deps BugHandlerDeps) {
	h := &BugHandler{
		logger:  deps.Logger,
		useCase: deps.UseCase,
	}
	r := g.Group("/bug")
	{
		r.POST("/create", auth.AuthOptional(), h.CreateBug)
		r.GET("/list", auth.AdminOnly(), h.List)
	}
}

type DeviceRequest struct {
	OS         string   `json:"os" binding:"required"`
	Platform   string   `json:"platform" binding:"required"`
	Resolution string   `json:"resolution" binding:"required"`
	Battery    *float64 `json:"battery" binding:"omitempty"`
}

type App struct {
	Build string   `json:"build" binding:"required"`
	Logs  []string `json:"logs" binding:"required"`
}

type BugCreateRequest struct {
	Title  string        `json:"title" binding:"required"`
	Desc   string        `json:"desc" binding:"required"`
	Tag    string        `json:"tag" binding:"required"`
	Device DeviceRequest `json:"device" binding:"required"`
	App    App           `json:"app" binding:"required"`
}

func (h *BugHandler) CreateBug(c *gin.Context) {
	userID, _ := helper.GetUserID(c)

	var uID *uint
	if userID > 0 {
		uID = utils.Ptr(uint(userID))
	}

	var req BugCreateRequest
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	err := h.useCase.Create.Handle(c.Request.Context(), bug.CreateBugCommand{
		Tag:    req.Tag,
		Title:  req.Title,
		Desc:   req.Desc,
		UserID: uID,
		Device: bug.DeviceInfoCommand{
			OS:         req.Device.OS,
			Platform:   req.Device.Platform,
			Resolution: req.Device.Resolution,
			Battery:    req.Device.Battery,
		},
		App: bug.ApplicationInfoCommand{
			Build: req.App.Build,
			Logs:  req.App.Logs,
		},
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusAccepted)
}

type BugRequestParams struct {
	pagination.Params
	Status *string `form:"status" binding:"omitempty"`
	Tag    *string `form:"tag" binding:"omitempty"`
}

func (h *BugHandler) List(c *gin.Context) {
	var req BugRequestParams
	if err := c.ShouldBindQuery(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	res, err := h.useCase.List.Handle(c.Request.Context(), bug.ListBugCommand{
		Tag:        req.Tag,
		Status:     req.Status,
		Pagination: req.Params,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, mapToListResponse(res))
}
