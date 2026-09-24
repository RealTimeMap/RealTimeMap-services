package kafka

import (
	"testing"
	"time"

	segmentio "github.com/segmentio/kafka-go"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
)

// TestSubscriptionEventResolvesSubscriber фиксирует, КОГО gamification-service
// видит в событии подписки.
//
// Это не деталь реализации, а ограничение для каталога достижений: событие
// несёт двух пользователей — подписавшегося (SubscriberID) и того, на кого
// подписались (TargetID), — но засчитывается оно ровно одному. Тест
// закрепляет, кому именно, чтобы достижения не разошлись со смыслом:
// «подпишитесь на N» работает, «получите N подписчиков» на этом событии
// выдать нельзя.
func TestSubscriptionEventResolvesSubscriber(t *testing.T) {
	requireKafka(t)

	const (
		topic        = "social-service.events"
		subscriberID = 1005
		targetID     = 9005
	)

	// Мета собирается ровно так, как это делает social-service:
	// UserID = подписавшийся, SourceID = адресат подписки.
	meta := producer.EventMeta{
		EventType: events.SubscriptionCreated,
		UserID:    "1005",
		SourceID:  "9005",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	from := tailOffset(t, topic)
	publish(t, topic, meta, events.NewSubscriptionCreated(events.SubscriptionPayload{
		SubscriberID:   subscriberID,
		TargetID:       targetID,
		SubscriberName: "integration test",
	}))

	msg := readFrom(t, topic, from, func(m segmentio.Message) bool {
		got, err := extractMeta(m)
		return err == nil && got.UserID == subscriberID
	})

	got, err := extractMeta(msg)
	if err != nil {
		t.Fatalf("extractMeta: %v", err)
	}

	if got.UserID != subscriberID {
		t.Errorf("UserID = %d, want %d (подписавшийся)", got.UserID, subscriberID)
	}
	if got.UserID == targetID {
		t.Error("событие засчитано адресату подписки — достижения «получите N подписчиков» стали бы выполнимы")
	}
}
