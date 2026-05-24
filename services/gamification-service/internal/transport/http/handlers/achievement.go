package handlers

import (
	"io"
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/achievement"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AchievementDeps struct {
	Service achievement.Service

	Logger *zap.Logger
}

type handler struct {
	service achievement.Service

	logger *zap.Logger
}

func InitAchievementHandler(g *gin.RouterGroup, deps AchievementDeps) {
	h := &handler{
		service: deps.Service,
		logger:  deps.Logger,
	}
	r := g.Group("/achievement")
	{
		r.POST("/create", auth.AdminOnly(), h.CreateAchievement)
		r.GET("/:userID", h.GetUserAchievements)
		r.GET("/:userID/nearest", h.GetUserNearlyAchievements)
	}
}

type AchievementRequest struct {
	Code         string `form:"code" binding:"required"`
	Title        string `form:"title" binding:"required"`
	Desc         string `form:"desc" binding:"required"`
	TriggerEvent string `form:"triggerEvent" binding:"required"`
	Threshold    uint   `form:"threshold" binding:"required"`
	RewardID     uint   `form:"rewardId" binding:"required"`
	NextID       *uint  `form:"nextId" binding:"omitempty"`
}

func (h *handler) CreateAchievement(c *gin.Context) {
	var req AchievementRequest
	if err := c.ShouldBind(&req); err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	var icon mediavalidator.PhotoInput
	fileHeader, err := c.FormFile("icon")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	data, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	icon = mediavalidator.PhotoInput{
		Data:     data,
		FileName: fileHeader.Filename,
	}

	achievement, err := h.service.CreateAchievement(c.Request.Context(), achievement.Input{
		Code:         req.Code,
		Title:        req.Title,
		Desc:         req.Desc,
		TriggerEvent: req.TriggerEvent,
		Threshold:    req.Threshold,
		RewardID:     req.RewardID,
		NextID:       req.NextID,
		Icon:         icon,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"achievement": ToAchievementResponse(achievement),
	})

}

func (h *handler) GetAchievements(c *gin.Context) {

}

func (h *handler) GetUserAchievements(c *gin.Context) {
	var params pagination.Params
	if err := c.ShouldBind(&params); err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	userID, err := middleware.ParsePathParams(c, "userID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	achs, total, err := h.service.GetAchievements(c.Request.Context(), userID, params)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	res := ToUserAchievementResponseList(achs)
	c.JSON(http.StatusOK, pagination.NewResponse(res, params, total))
}

func (h *handler) GetUserNearlyAchievements(c *gin.Context) {
	userID, err := middleware.ParsePathParams(c, "userID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	items, err := h.service.GetNearestAchievements(c.Request.Context(), userID)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": ToNearestAchievementResponseList(items),
	})
}
