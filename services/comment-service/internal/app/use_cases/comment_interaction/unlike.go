package comment_interaction

import (
	"context"

	"go.uber.org/zap"
)

type UnlikeCommentHandler struct {
	unliker CommentUnliker

	logger *zap.Logger
}

func NewUnlikeCommentHandler(unliker CommentUnliker, logger *zap.Logger) *UnlikeCommentHandler {
	return &UnlikeCommentHandler{unliker: unliker, logger: logger}
}

func (h *UnlikeCommentHandler) Handle(ctx context.Context, cmd ReactionCommand) (ReactionResult, error) {
	h.logger.Info("start commentInteraction.UnlikeCommentHandler.Handle")

	count, liked, err := h.unliker.Unlike(ctx, cmd.CommentID, cmd.UserID)
	if err != nil {
		return ReactionResult{}, err
	}
	return toReactionResult(count, liked), nil
}
