package handlers

import (
	"net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_interaction"
	dto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/mark"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AccrualDeps struct {
	UseCases *mark_interaction.Application

	Logger *zap.Logger
}

type accrualHandler struct {
	useCases *mark_interaction.Application
	logger   *zap.Logger
}

func InitAccrualHandler(g *gin.RouterGroup, deps AccrualDeps) {
	h := &accrualHandler{useCases: deps.UseCases, logger: deps.Logger}
	accrualGroup := g.Group("/accrual")
	{
		accrualGroup.POST("/:markID/share", h.IncreaseShare)
		accrualGroup.POST("/:markID/like", auth.AuthRequired(), h.Like)
		accrualGroup.DELETE("/:markID/like", auth.AuthRequired(), h.Unlike)
		accrualGroup.GET("/:markID/stat", auth.AuthOptional(), h.Stat)
	}
}

func (h *accrualHandler) IncreaseShare(c *gin.Context) {
	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	result, err := h.useCases.CreateShare.Handle(c.Request.Context(), mark_interaction.ShareCreateCommand{
		MarkID: markID,
		UserID: 0, // TODO При переносе на более сложную модель хранения поделившихся объектов поменять
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": result.Count,
	})
}

func (h *accrualHandler) Like(c *gin.Context) {
	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	result, err := h.useCases.LikeMark.Handle(c.Request.Context(), mark_interaction.ReactionCommand{
		MarkID: markID,
		UserID: uint(userID),
	})

	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewReactionResponse(result))
}

func (h *accrualHandler) Unlike(c *gin.Context) {
	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	result, err := h.useCases.UnlikeMark.Handle(c.Request.Context(), mark_interaction.ReactionCommand{
		MarkID: markID,
		UserID: uint(userID),
	})

	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewReactionResponse(result))
}

func (h *accrualHandler) Stat(c *gin.Context) {
	markID, err := middleware.ParsePathParams(c, "markID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	// userID опционален: без токена возвращаем статистику с isLiked=false.
	userID, _ := helper.GetUserID(c)

	result, err := h.useCases.GetStat.Handle(c.Request.Context(), mark_interaction.StatCommand{
		MarkID: markID,
		UserID: uint(userID),
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewMarkStatResponse(result))
}
