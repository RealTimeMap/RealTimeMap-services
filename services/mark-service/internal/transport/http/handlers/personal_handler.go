package handlers

import (
	"mime/multipart"
	"net/http"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/paulmach/orb"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	usecase "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/personal"
)

type PersonalDeps struct {
	UseCase *usecase.Application
	Logger  *zap.Logger
}

type personalHandler struct {
	useCase *usecase.Application

	logger *zap.Logger
}

func InitPersonalMarkHandler(g *gin.RouterGroup, deps PersonalDeps) {
	h := &personalHandler{
		useCase: deps.UseCase,
		logger:  deps.Logger,
	}
	r := g.Group("/personal")
	{
		r.POST("/create", auth.AuthRequired(), h.Create)
		r.GET("/sync", auth.AuthRequired(), h.Sync)
	}
}

type CreatePersonalMarkRequest struct {
	Title       string  `form:"title" binding:"required"`
	Description *string `form:"description" binding:"-"`
	Category    string  `form:"category" binding:"required"`
	Icon        string  `form:"icon" binding:"required"`
	Color       string  `form:"color" binding:"required"`
	IsVisible   bool    `form:"isVisible"`

	Longitude float64 `form:"longitude" binding:"required,longitude"`
	Latitude  float64 `form:"latitude" binding:"required,latitude"`

	GroupsIds []uint                  `form:"groupsId" binding:"required"`
	Photos    []*multipart.FileHeader `form:"photos" binding:"-"`
}

func (h *personalHandler) Create(c *gin.Context) {
	var req CreatePersonalMarkRequest

	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	photos, err := processPhotoUploads(req.Photos)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	obj, err := h.useCase.Create.Handle(c.Request.Context(), usecase.CreatePersonalMarkCommand{
		UserID:      uint(userInfo.UserID),
		Geom:        types.Point{Point: orb.Point{req.Longitude, req.Latitude}},
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
		Color:       req.Color,
		Icon:        req.Icon,
		IsVisible:   req.IsVisible,
		GroupsIds:   req.GroupsIds,
		Photos:      photos,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusCreated, obj)
}

type SyncRequest struct {
	Since *uint `form:"since" binding:"omitempty"`
	Limit int   `form:"limit" binding:"gt=0"`
}

func (h *personalHandler) Sync(c *gin.Context) {
	req := SyncRequest{
		Limit: 100,
	}
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	obj, err := h.useCase.Sync.Hanlde(c.Request.Context(), uint(userInfo.UserID), usecase.SyncCommand{
		Since: req.Since,
		Limit: req.Limit,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(200, obj)
}
