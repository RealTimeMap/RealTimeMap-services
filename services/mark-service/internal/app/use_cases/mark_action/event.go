package mark_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
)

// EventPublisher — порт для публикации доменных событий во внешнюю шину (Kafka).
// Интерфейс объявлен на стороне потребителя; *producer.Producer удовлетворяет его напрямую.
// event сериализуется целиком в тело сообщения (envelope + payload).
type EventPublisher interface {
	PublishWithMeta(ctx context.Context, meta producer.EventMeta, payload any) error
}
