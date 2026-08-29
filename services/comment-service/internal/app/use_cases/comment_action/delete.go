package comment_action

import (
	"context"

	"go.uber.org/zap"
)

type CommentRemover interface {
	SoftDelete(ctx context.Context, userID, commentID uint) error
}

type DeleteCommentCommand struct {
	UserID    uint
	CommentID uint
}

type DeleteCommentHandler struct {
	remover CommentRemover

	logger *zap.Logger
}

func NewDeleteCommentHandler(remover CommentRemover, logger *zap.Logger) *DeleteCommentHandler {
	return &DeleteCommentHandler{
		remover: remover,
		logger:  logger,
	}
}

func (h *DeleteCommentHandler) Handle(ctx context.Context, cmd DeleteCommentCommand) error {
	h.logger.Info("start commentUseCases.DeleteCommentHandler.Handle")
	return h.remover.SoftDelete(ctx, cmd.UserID, cmd.CommentID)
}
