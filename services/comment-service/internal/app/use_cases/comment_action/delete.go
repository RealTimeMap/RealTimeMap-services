package comment_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"go.uber.org/zap"
)

type CommentRemover interface {
	SoftDelete(ctx context.Context, userID, commentID uint) (*comment.Comment, error)
}

type DeleteCommentCommand struct {
	UserID    uint
	CommentID uint
}

type DeleteCommentHandler struct {
	remover   CommentRemover
	publisher EventPublisher

	logger *zap.Logger
}

func NewDeleteCommentHandler(remover CommentRemover, publisher EventPublisher, logger *zap.Logger) *DeleteCommentHandler {
	return &DeleteCommentHandler{
		remover:   remover,
		publisher: publisher,
		logger:    logger,
	}
}

func (h *DeleteCommentHandler) Handle(ctx context.Context, cmd DeleteCommentCommand) error {
	h.logger.Info("start commentUseCases.DeleteCommentHandler.Handle")

	deleted, err := h.remover.SoftDelete(ctx, cmd.UserID, cmd.CommentID)
	if err != nil {
		return err
	}

	publishAsync(h.logger, events.CommentDeleted, func(ctx context.Context) error {
		return h.publisher.PublishCommentDeleted(ctx, deleted)
	})

	return nil
}
