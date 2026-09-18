package handlers

import (
	"mime/multipart"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
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
		r.GET("/:markID", auth.AuthRequired(), h.Get)
		r.PATCH("/:markID", auth.AuthRequired(), h.Update)
		r.DELETE("/:markID", auth.AuthRequired(), h.Delete)
	}
}

type CreatePersonalMarkRequest struct {
	Title       string  `form:"title" binding:"required"`
	Description *string `form:"description" binding:"-"`
	Icon        string  `form:"icon" binding:"required"`
	Color       string  `form:"color" binding:"required"`
	IsVisible   bool    `form:"isVisible"`

	Longitude float64 `form:"longitude" binding:"required,longitude"`
	Latitude  float64 `form:"latitude" binding:"required,latitude"`

	GroupsIds []string                `form:"groupsIds" binding:"required"`
	Photos    []*multipart.FileHeader `form:"photos" binding:"-"`
}

// parseGroupIDs разбирает идентификаторы групп из формы.
func parseGroupIDs(raw []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(raw))

	for _, item := range raw {
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			id, err := uuid.Parse(part)
			if err != nil {
				return nil, apperror.NewFieldValidationError(
					"groupsIds",
					"groupsIds must contain uuid values",
					"value_error.uuid",
					part,
				)
			}
			out = append(out, id)
		}
	}

	return out, nil
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

	groupIDs, err := parseGroupIDs(req.GroupsIds)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
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
		Color:       req.Color,
		Icon:        req.Icon,
		IsVisible:   req.IsVisible,
		GroupsIds:   groupIDs,
		Photos:      photos,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusCreated, obj)
}

const defaultSyncLimit = 100

type SyncRequest struct {
	Since *uint `form:"since" binding:"omitempty"`
	Limit int   `form:"limit" binding:"omitempty,gt=0,lte=500"`
}

func (h *personalHandler) Sync(c *gin.Context) {
	req := SyncRequest{}

	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	if req.Limit == 0 {
		req.Limit = defaultSyncLimit
	}

	obj, err := h.useCase.Sync.Handle(c.Request.Context(), uint(userInfo.UserID), usecase.SyncCommand{
		Since: req.Since,
		Limit: req.Limit,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, obj)
}

func (h *personalHandler) Get(c *gin.Context) {
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	obj, err := h.useCase.Get.Handle(c.Request.Context(), usecase.GetPersonalMarkQuery{
		MarkID: markID,
		UserID: uint(userInfo.UserID),
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, obj)
}

// UpdatePersonalMarkRequest — частичное обновление: поля указатели, чтобы
// отличить "не передано" от пустой строки и снятого флага.
type UpdatePersonalMarkRequest struct {
	Title       *string `form:"title" binding:"omitempty"`
	Description *string `form:"description" binding:"-"`
	Icon        *string `form:"icon" binding:"omitempty"`
	Color       *string `form:"color" binding:"omitempty"`
	IsVisible   *bool   `form:"isVisible" binding:"omitempty"`

	Longitude *float64 `form:"longitude" binding:"omitempty,longitude"`
	Latitude  *float64 `form:"latitude" binding:"omitempty,latitude"`

	// Строками по той же причине, что и в CreatePersonalMarkRequest.
	GroupsIds []string `form:"groupsIds" binding:"-"`

	PhotosToDelete []string                `form:"photosToDelete" binding:"-"`
	Photos         []*multipart.FileHeader `form:"photos" binding:"-"`
}

func (h *personalHandler) Update(c *gin.Context) {
	var req UpdatePersonalMarkRequest

	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	markID, err := middleware.ParsePathParams(c, "markID")
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

	// Координаты имеют смысл только парой: сдвиг по одной оси оставил бы метку
	// в точке, которую клиент не запрашивал.
	var geom *types.Point
	if req.Longitude != nil && req.Latitude != nil {
		geom = &types.Point{Point: orb.Point{*req.Longitude, *req.Latitude}}
	} else if req.Longitude != nil || req.Latitude != nil {
		errorhandler.HandleError(c, apperror.NewFieldValidationError(
			"longitude",
			"longitude and latitude must be provided together",
			"value_error.geom.incomplete",
			nil,
		), h.logger)
		return
	}

	groupIDs, err := parseGroupIDs(req.GroupsIds)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	obj, err := h.useCase.Update.Handle(c.Request.Context(), usecase.UpdatePersonalMarkCommand{
		MarkID:         markID,
		UserID:         uint(userInfo.UserID),
		Geom:           geom,
		Title:          req.Title,
		Description:    req.Description,
		Color:          req.Color,
		Icon:           req.Icon,
		IsVisible:      req.IsVisible,
		GroupsIds:      groupIDs,
		PhotosToDelete: req.PhotosToDelete,
		Photos:         photos,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, obj)
}

func (h *personalHandler) Delete(c *gin.Context) {
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	err = h.useCase.Delete.Handle(c.Request.Context(), usecase.DeletePersonalMarkCommand{
		MarkID: markID,
		UserID: uint(userInfo.UserID),
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.Status(http.StatusNoContent)
}
