package handlers

import (
	"net/http"
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	ctxhelper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_action"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type commentHandler struct {
	useCases *comment_action.Application
	logger   *zap.Logger
}

type CommentDeps struct {
	UseCases *comment_action.Application
	Logger   *zap.Logger
}

func InitCommentHandler(g *gin.RouterGroup, deps CommentDeps) {
	h := &commentHandler{useCases: deps.UseCases, logger: deps.Logger}
	r := g.Group("")
	{
		r.POST("/comments", auth.AuthRequired(), h.CreateComment)

		r.GET("/:id/comments", h.GetComments)

		r.GET("/:id/comments/:parentID/replies", h.GetReplies)

		r.DELETE("/:id/comments", auth.AuthRequired(), h.DeleteComment)

		r.PATCH("/:id/comments", auth.AuthRequired(), h.UpdateComment)
	}
}

func (h *commentHandler) CreateComment(c *gin.Context) {
	var req dto.CommentRequest

	userData, err := ctxhelper.GetUserInfo(c)
	if err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, err := h.useCases.CreateComment.Handle(c.Request.Context(), comment_action.CreateCommentCommand{
		Content:    req.Content,
		EntityType: req.Entity,
		EntityID:   req.EntityID,
		ParentID:   req.ParentID,
		UserID:     uint(userData.UserID),
		Username:   userData.UserName,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusCreated, dto.NewCommentResponse(res))
}

func (h *commentHandler) GetComments(c *gin.Context) {
	var req dto.CommentParams

	entityID, err := h.parseIDParam(c, "id")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err = c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	page, err := h.useCases.GetComments.Handle(c.Request.Context(), req.ToFilter(entityID, nil))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewCursorPaginateResponse(page))
}

func (h *commentHandler) GetReplies(c *gin.Context) {
	var req dto.CommentParams

	entityID, err := h.parseIDParam(c, "id")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	parentID, err := h.parseIDParam(c, "parentID")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err = c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	page, err := h.useCases.GetComments.Handle(c.Request.Context(), req.ToFilter(entityID, &parentID))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewCursorPaginateResponse(page))
}

func (h *commentHandler) DeleteComment(c *gin.Context) {
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

	err = h.useCases.DeleteComment.Handle(c.Request.Context(), comment_action.DeleteCommentCommand{
		UserID:    uint(userID),
		CommentID: commentID,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *commentHandler) UpdateComment(c *gin.Context) {
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

	var req dto.CommentUpdateRequest
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, err := h.useCases.UpdateComment.Handle(c.Request.Context(), comment_action.UpdateCommentCommand{
		Content:   req.Content,
		UserID:    uint(userID),
		CommentID: commentID,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewCommentResponse(res))
}

func (h *commentHandler) parseIDParam(c *gin.Context, param string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		return 0, apperror.NewFieldValidationError("id", "id must be a number", "value_error", c.Param(param))
	}
	return uint(id), nil
}
