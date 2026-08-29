package comment_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"go.uber.org/zap"
)

type CommentGetter interface {
	GetComments(ctx context.Context, filters comment.CommentFilter) ([]*comment.Comment, bool, error)
}

type GetCommentsHandler struct {
	getter   CommentGetter
	provider ProfileProvider

	logger *zap.Logger
}

func NewGetCommentsHandler(getter CommentGetter, provider ProfileProvider, logger *zap.Logger) *GetCommentsHandler {
	return &GetCommentsHandler{
		getter:   getter,
		provider: provider,
		logger:   logger,
	}
}

func (h *GetCommentsHandler) Handle(ctx context.Context, filter comment.CommentFilter) (CursorPage, error) {
	h.logger.Info("start commentUseCases.GetCommentsHandler.Handle")

	comments, hasMore, err := h.getter.GetComments(ctx, filter)
	if err != nil {
		return CursorPage{}, err
	}

	attachAuthors(ctx, h.provider, h.logger, comments)
	return CursorPage{Items: toMultiCommentResult(comments), HasMore: hasMore}, nil
}
