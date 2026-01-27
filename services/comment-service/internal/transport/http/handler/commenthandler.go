package handler

import (
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
		r.GET("/:id/comments", h.GetComments)
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
	entityID, err := strconv.ParseUint(c.Param("id"), 10, 64)

	if err != nil {
		err = apperror.NewFieldValidationError("id", "id must be a number", "value_error", c.Param("id"))
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	comments, hasMore, err := h.service.GetComments(c.Request.Context(), req.ToFilter(uint(entityID)))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(200, dto.NewCursorPaginateResponse(comments, hasMore))
}
