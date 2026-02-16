package handler

import (
	"net/http"
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service/comment"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *comment.Service

	logger *zap.Logger
}

func NewCommentRoute(g *gin.RouterGroup, service *comment.Service, logger *zap.Logger) {
	h := &Handler{service: service, logger: logger}
	r := g.Group("")
	{
		r.POST("/comments", auth.AuthRequired(), h.CreateComment)
		r.POST("/comments/", auth.AuthRequired(), h.CreateComment)

		r.GET("/:id/comments", h.GetComments)
		r.GET("/:id/comments/", h.GetComments)

		r.GET("/:id/comments/:parentID/replies", h.GetReplies)
		r.GET("/:id/comments/:parentID/replies/", h.GetReplies)

		r.DELETE("/:id/comments", auth.AuthRequired(), h.DeleteComment)
		r.DELETE("/:id/comments/", auth.AuthRequired(), h.DeleteComment)

		r.PATCH("/:id/comments", auth.AuthRequired(), h.UpdateComment)
		r.PATCH("/:id/comments/", auth.AuthRequired(), h.UpdateComment)
	}
}

func (h *Handler) CreateComment(c *gin.Context) {
	var req dto.CommentRequest

	userID, err := context.GetUserID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	newComment, err := h.service.Create(c.Request.Context(), comment.CreateInput{Content: req.Content, ParentID: req.ParentID, EntityID: req.EntityID, EntityType: req.Entity}, uint(userID))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(201, dto.NewCommentResponse(newComment))
}

func (h *Handler) GetComments(c *gin.Context) {
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

	comments, hasMore, err := h.service.GetComments(c.Request.Context(), req.ToFilter(entityID, nil))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(200, dto.NewCursorPaginateResponse(comments, hasMore))
}

func (h *Handler) GetReplies(c *gin.Context) {
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

	replies, hasMore, err := h.service.GetComments(c.Request.Context(), req.ToFilter(entityID, &parentID))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(200, dto.NewCursorPaginateResponse(replies, hasMore))
}

func (h *Handler) DeleteComment(c *gin.Context) {
	commentID, err := h.parseIDParam(c, "id")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	userID, err := context.GetUserID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	err = h.service.SoftDelete(c.Request.Context(), uint(userID), commentID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.Status(http.StatusNoContent)

}

func (h *Handler) UpdateComment(c *gin.Context) {
	commentID, err := h.parseIDParam(c, "id")
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	userID, err := context.GetUserID(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	var req dto.CommentUpdateRequest
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	uComment, err := h.service.UpdateComment(c.Request.Context(), comment.UpdateInput{Content: req.Content}, uint(userID), commentID)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewCommentResponse(uComment))
}

func (h *Handler) parseIDParam(c *gin.Context, param string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(param), 10, 64)
	if err != nil {
		return 0, apperror.NewFieldValidationError("id", "id must be a number", "value_error", c.Param("id"))
	}
	return uint(id), nil
}
