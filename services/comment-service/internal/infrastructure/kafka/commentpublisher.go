package kafka

import (
	"context"
	"strconv"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app/use_cases/comment_action"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"go.uber.org/zap"
)

type CommentPublisher struct {
	producer *producer.Producer
	logger   *zap.Logger
}

func NewCommentPublisher(p *producer.Producer, logger *zap.Logger) comment_action.EventPublisher {
	return &CommentPublisher{producer: p, logger: logger}
}

func (p *CommentPublisher) PublishCommentCreated(ctx context.Context, c *comment.Comment) error {
	payload := events.NewCommentPayload(
		c.ID,
		c.UserID,
		c.EntityID,
		string(c.EntityType),
		c.ParentID,
		c.Content,
	)

	event := events.NewCommentCreated(payload)

	if err := p.producer.PublishWithMeta(ctx, p.buildMeta(events.CommentCreated, c), event); err != nil {
		p.logger.Error("failed to publish comment.created",
			zap.Uint("commentID", c.ID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("published comment.created", zap.Uint("commentID", c.ID))
	return nil
}

func (p *CommentPublisher) buildMeta(eventType string, c *comment.Comment) producer.EventMeta {
	return producer.EventMeta{
		EventType: eventType,
		UserID:    strconv.FormatUint(uint64(c.UserID), 10),
		SourceID:  strconv.FormatUint(uint64(c.ID), 10),
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (p *CommentPublisher) Close() error {
	return p.producer.Close()
}
