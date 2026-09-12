package comment_action

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"go.uber.org/zap"
)

// publishTimeout ограничивает фоновую публикацию: событие уходит уже после
// ответа пользователю, и зависший вызов не должен держать горутину вечно.
const publishTimeout = 5 * time.Second

// EventPublisher — порт для публикации доменных событий комментариев во внешнюю шину.
// Интерфейс объявлен на стороне потребителя; kafka.CommentPublisher удовлетворяет его.
type EventPublisher interface {
	PublishCommentCreated(ctx context.Context, c *comment.Comment) error
	PublishCommentUpdated(ctx context.Context, c *comment.Comment) error
	PublishCommentDeleted(ctx context.Context, c *comment.Comment) error
}

// NoOpEventPublisher — заглушка на случай выключенной шины.
type NoOpEventPublisher struct{}

func (NoOpEventPublisher) PublishCommentCreated(context.Context, *comment.Comment) error {
	return nil
}

func (NoOpEventPublisher) PublishCommentUpdated(context.Context, *comment.Comment) error {
	return nil
}

func (NoOpEventPublisher) PublishCommentDeleted(context.Context, *comment.Comment) error {
	return nil
}

// publishAsync публикует событие в фоне.
//
// Сбой шины не влияет на результат операции: комментарий уже сохранён, и
// откатывать его из-за недоступного брокера нельзя. Цена — потеря события при
// отказе Kafka; чинится outbox-таблицей, когда это станет критично.
//
// Контекст берётся новый, а не производный от запросного: тот отменяется
// сразу после ответа клиенту, и публикация не успевала бы уйти.
func publishAsync(logger *zap.Logger, eventType string, publish func(context.Context) error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
		defer cancel()

		if err := publish(ctx); err != nil {
			logger.Warn("failed to publish comment event",
				zap.String("event_type", eventType),
				zap.Error(err),
			)
		}
	}()
}
