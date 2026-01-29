package service

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
)

type EventPublisher interface {
	PublishCommentCreated(ctx context.Context, comment *model.Comment) error
}

type NoOpEventPublisher struct{}

func (n *NoOpEventPublisher) PublishCommentCreated(ctx context.Context, comment *model.Comment) error {
	return nil
}

func (n *NoOpEventPublisher) PublishCommentUpdated(ctx context.Context, comment *model.Comment) error {
	return nil
}

func (n *NoOpEventPublisher) PublishCommentDeleted(ctx context.Context, comment *model.Comment) error {
	return nil
}
