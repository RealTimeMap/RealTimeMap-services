package handlers

import (
	"net/http"
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	ctxhelper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_interaction"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type reactionHandler struct {
	useCases *comment_interaction.Application
	logger   *zap.Logger
}

type ReactionDeps struct {
	UseCases *comment_interaction.Application
	Logger   *zap.Logger
}

func InitReactionHandler(g *gin.RouterGroup, deps ReactionDeps) {
	h := &reactionHandler{useCases: deps.UseCases, logger: deps.Logger}
	r := g.Group("")
	{
		r.POST("/:id/comments/reaction", auth.AuthRequired(), h.Like)

		r.DELETE("/:id/comments/reaction", auth.AuthRequired(), h.Unlike)
	}
}

func (h *reactionHandler) Like(c *gin.Context) {
	commentID, err := h.parseIDParam(c, "id")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	userID, err := ctxhelper.GetUserID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	res, err := h.useCases.LikeComment.Handle(c.Request.Context(), comment_interaction.ReactionCommand{
		CommentID: commentID,
		UserID:    uint(userID),
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewReactionResponse(res))
}

func (h *reactionHandler) Unlike(c *gin.Context) {
	commentID, err := h.parseIDParam(c, "id")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	userID, err := ctxhelper.GetUserID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	res, err := h.useCases.UnlikeComment.Handle(c.Request.Context(), comment_interaction.ReactionCommand{
		CommentID: commentID,
		UserID:    uint(userID),
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewReactionResponse(res))
}

func (h *reactionHandler) parseIDParam(c *gin.Context, param string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		return 0, apperror.NewFieldValidationError("id", "id must be a number", "value_error", c.Param(param))
	}
	return uint(id), nil
}
