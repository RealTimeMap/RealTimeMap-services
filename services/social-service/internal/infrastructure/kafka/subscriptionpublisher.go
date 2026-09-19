package kafka

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/subscription"
)

// SubscriptionPublisher публикует события подписок в Kafka.
type SubscriptionPublisher struct {
	producer *producer.Producer
	topic    string
	logger   *zap.Logger
}

func NewSubscriptionPublisher(p *producer.Producer, topic string, logger *zap.Logger) subscription.EventPublisher {
	return &SubscriptionPublisher{producer: p, topic: topic, logger: logger}
}

// PublishSubscriptionCreated отправляет событие о новой подписке.
//
// Ключ партиционирования — подписавшийся (UserID в мете): подписки одного
// пользователя идут в одну партицию и не обгоняют друг друга. Адресатом
// уведомления при этом остаётся TargetID — он в payload.
func (p *SubscriptionPublisher) PublishSubscriptionCreated(ctx context.Context, e subscription.SubscriptionCreated) error {
	payload := events.SubscriptionPayload{
		SubscriberID:   e.SubscriberID,
		TargetID:       e.TargetID,
		SubscriberName: e.SubscriberName,
	}

	meta := producer.EventMeta{
		EventType: events.SubscriptionCreated,
		UserID:    strconv.FormatUint(uint64(e.SubscriberID), 10),
		SourceID:  strconv.FormatUint(uint64(e.TargetID), 10),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := p.producer.PublishWithMeta(ctx, meta, events.NewSubscriptionCreated(payload)); err != nil {
		p.logger.Error("failed to publish subscription event",
			zap.String("event_type", events.SubscriptionCreated),
			zap.Uint("subscriber_id", e.SubscriberID),
			zap.Uint("target_id", e.TargetID),
			zap.Error(err),
		)
		return err
	}

	p.logger.Debug("published subscription event",
		zap.String("event_type", events.SubscriptionCreated),
		zap.String("topic", p.topic),
		zap.Uint("subscriber_id", e.SubscriberID),
		zap.Uint("target_id", e.TargetID),
	)
	return nil
}
