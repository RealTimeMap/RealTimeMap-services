package comment_interaction

import (
	"context"

	"go.uber.org/zap"
)

type ReactionCommand struct {
	CommentID uint
	UserID    uint
}

// CommentLiker ставит лайк на комментарий.
type CommentLiker interface {
	Like(ctx context.Context, commentID, userID uint) (int64, bool, error)
}

// CommentUnliker снимает лайк с комментария.
type CommentUnliker interface {
	Unlike(ctx context.Context, commentID, userID uint) (int64, bool, error)
}

type LikeCommentHandler struct {
	liker CommentLiker

	logger *zap.Logger
}

func NewLikeCommentHandler(liker CommentLiker, logger *zap.Logger) *LikeCommentHandler {
	return &LikeCommentHandler{liker: liker, logger: logger}
}

func (h *LikeCommentHandler) Handle(ctx context.Context, cmd ReactionCommand) (ReactionResult, error) {
	h.logger.Info("start commentInteraction.LikeCommentHandler.Handle")

	count, liked, err := h.liker.Like(ctx, cmd.CommentID, cmd.UserID)
	if err != nil {
		return ReactionResult{}, err
	}
	return toReactionResult(count, liked), nil
}
