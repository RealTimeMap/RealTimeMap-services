package kafka

import (
	"context"
	"strconv"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service"
	"go.uber.org/zap"
)

type CommentPublisher struct {
	producer *producer.Producer
	logger   *zap.Logger
}

func NewCommentPublisher(p *producer.Producer, logger *zap.Logger) service.EventPublisher {
	return &CommentPublisher{producer: p, logger: logger}
}

func (p *CommentPublisher) PublishCommentCreated(ctx context.Context, comment *model.Comment) error {
	payload := events.NewCommentPayload(
		comment.ID,
		comment.UserID,
		comment.EntityID,
		string(comment.EntityType),
		comment.ParentID,
		comment.Content,
	)

	event := events.NewCommentCreated(payload)

	if err := p.producer.PublishWithMeta(ctx, p.buildMeta(events.CommentCreated, comment), event); err != nil {
		p.logger.Error("failed to publish comment.created",
			zap.Uint("commentID", comment.ID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("published comment.created", zap.Uint("commentID", comment.ID))
	return nil
}

func (p *CommentPublisher) buildMeta(eventType string, comment *model.Comment) producer.EventMeta {
	return producer.EventMeta{
		EventType: eventType,
		UserID:    strconv.FormatUint(uint64(comment.UserID), 10),
		SourceID:  strconv.FormatUint(uint64(comment.ID), 10),
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (p *CommentPublisher) Close() error {
	return p.producer.Close()
}
