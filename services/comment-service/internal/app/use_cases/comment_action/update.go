package comment_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"go.uber.org/zap"
)

type CommentUpdater interface {
	UpdateComment(ctx context.Context, params comment.UpdateParams, userID, commentID uint) (*comment.Comment, error)
}

type UpdateCommentCommand struct {
	Content   string
	UserID    uint
	CommentID uint
}

type UpdateCommentHandler struct {
	updater   CommentUpdater
	provider  ProfileProvider
	publisher EventPublisher

	logger *zap.Logger
}

func NewUpdateCommentHandler(updater CommentUpdater, provider ProfileProvider, publisher EventPublisher, logger *zap.Logger) *UpdateCommentHandler {
	return &UpdateCommentHandler{
		updater:   updater,
		provider:  provider,
		publisher: publisher,
		logger:    logger,
	}
}

func (h *UpdateCommentHandler) Handle(ctx context.Context, cmd UpdateCommentCommand) (CommentResult, error) {
	h.logger.Info("start commentUseCases.UpdateCommentHandler.Handle")

	updated, err := h.updater.UpdateComment(ctx, comment.UpdateParams{Content: cmd.Content}, cmd.UserID, cmd.CommentID)
	if err != nil {
		return CommentResult{}, err
	}

	publishAsync(h.logger, events.CommentUpdated, func(ctx context.Context) error {
		return h.publisher.PublishCommentUpdated(ctx, updated)
	})

	attachAuthors(ctx, h.provider, h.logger, []*comment.Comment{updated})
	return toCommentResult(updated), nil
}
