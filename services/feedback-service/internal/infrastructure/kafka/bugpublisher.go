// Package kafka публикует события feedback-service в шину.
package kafka

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	bugcases "github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app/use_cases/bug"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"go.uber.org/zap"
)

type BugPublisher struct {
	producer *producer.Producer
	logger   *zap.Logger
}

func NewBugPublisher(p *producer.Producer, logger *zap.Logger) bugcases.EventPublisher {
	return &BugPublisher{producer: p, logger: logger}
}

// PublishBugConfirmed сообщает, что баг из отчёта пользователя подтверждён.
//
// user_id и source_id дублируются в headers: gamification-service берёт
// мету из тела и откатывается на заголовки, если тело её не несёт.
func (p *BugPublisher) PublishBugConfirmed(ctx context.Context, b *bug.Model) error {
	if b.UserID == nil {
		return errors.New("bug has no author to credit")
	}

	payload := events.BugPayload{
		BugID:  b.ID,
		UserID: *b.UserID,
		Tag:    string(b.Tag),
	}
	meta := producer.EventMeta{
		EventType: events.BugConfirmed,
		UserID:    strconv.FormatUint(uint64(*b.UserID), 10),
		SourceID:  strconv.FormatUint(uint64(b.ID), 10),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := p.producer.PublishWithMeta(ctx, meta, events.NewBugConfirmed(payload)); err != nil {
		p.logger.Error("failed to publish bug event",
			zap.String("event_type", events.BugConfirmed),
			zap.Uint("bug_id", b.ID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("published bug event",
		zap.String("event_type", events.BugConfirmed),
		zap.Uint("bug_id", b.ID),
	)
	return nil
}

func (p *BugPublisher) Close() error {
	return p.producer.Close()
}
