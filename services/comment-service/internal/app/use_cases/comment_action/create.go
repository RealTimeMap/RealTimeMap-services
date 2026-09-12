package comment_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"go.uber.org/zap"
)

type CommentCreator interface {
	Create(ctx context.Context, params comment.CreateParams, userID uint, username string) (*comment.Comment, error)
}

type CreateCommentCommand struct {
	Content    string
	EntityType string
	EntityID   uint
	ParentID   *uint

	UserID   uint
	Username string
}

type CreateCommentHandler struct {
	creator   CommentCreator
	provider  ProfileProvider
	publisher EventPublisher

	logger *zap.Logger
}

func NewCreateCommentHandler(creator CommentCreator, provider ProfileProvider, publisher EventPublisher, logger *zap.Logger) *CreateCommentHandler {
	return &CreateCommentHandler{
		creator:   creator,
		provider:  provider,
		publisher: publisher,
		logger:    logger,
	}
}

func (h *CreateCommentHandler) Handle(ctx context.Context, cmd CreateCommentCommand) (CommentResult, error) {
	h.logger.Info("start commentUseCases.CreateCommentHandler.Handle")

	newComment, err := h.creator.Create(ctx, comment.CreateParams{
		Content:    cmd.Content,
		EntityType: cmd.EntityType,
		EntityID:   cmd.EntityID,
		ParentID:   cmd.ParentID,
	}, cmd.UserID, cmd.Username)
	if err != nil {
		return CommentResult{}, err
	}

	publishAsync(h.logger, events.CommentCreated, func(ctx context.Context) error {
		return h.publisher.PublishCommentCreated(ctx, newComment)
	})

	attachAuthors(ctx, h.provider, h.logger, []*comment.Comment{newComment})
	return toCommentResult(newComment), nil
}
