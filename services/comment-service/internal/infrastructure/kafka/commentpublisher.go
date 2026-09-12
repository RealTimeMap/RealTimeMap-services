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
	return p.publish(ctx, events.CommentCreated, c, events.NewCommentCreated)
}

func (p *CommentPublisher) PublishCommentUpdated(ctx context.Context, c *comment.Comment) error {
	return p.publish(ctx, events.CommentUpdated, c, events.NewCommentUpdated)
}

// PublishCommentDeleted сообщает об удалении комментария.
//
// Текст в событие не попадает: NewCommentDeleted его вычищает, чтобы удалённый
// комментарий не оставался читаемым в топике до истечения retention.
func (p *CommentPublisher) PublishCommentDeleted(ctx context.Context, c *comment.Comment) error {
	return p.publish(ctx, events.CommentDeleted, c, events.NewCommentDeleted)
}

// publish собирает событие и отправляет его с метой в headers.
//
// build передаётся параметром, потому что конструкторы событий различаются не
// только типом в конверте: NewCommentDeleted дополнительно чистит payload.
func (p *CommentPublisher) publish(
	ctx context.Context,
	eventType string,
	c *comment.Comment,
	build func(events.CommentPayload) events.CommentEvent,
) error {
	var parentUserID *uint
	if c.Parent != nil {
		id := c.Parent.UserID
		parentUserID = &id
	}

	payload := events.NewCommentPayload(
		c.ID,
		c.UserID,
		c.EntityID,
		string(c.EntityType),
		c.Username,
		c.ParentID,
		parentUserID,
		c.Content,
	)

	if err := p.producer.PublishWithMeta(ctx, p.buildMeta(eventType, c), build(payload)); err != nil {
		p.logger.Error("failed to publish comment event",
			zap.String("event_type", eventType),
			zap.Uint("commentID", c.ID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("published comment event",
		zap.String("event_type", eventType),
		zap.Uint("commentID", c.ID),
	)
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
