package bug

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"go.uber.org/zap"
)

// publishTimeout ограничивает фоновую публикацию: зависший брокер не
// должен копить горутины.
const publishTimeout = 5 * time.Second

// EventPublisher отправляет события о багах в шину. По ним
// gamification-service начисляет автору отчёта опыт и достижения.
type EventPublisher interface {
	PublishBugConfirmed(ctx context.Context, b *bug.Model) error
}

// NoOpEventPublisher — заглушка на случай выключенной шины.
type NoOpEventPublisher struct{}

func (NoOpEventPublisher) PublishBugConfirmed(context.Context, *bug.Model) error {
	return nil
}

// publishAsync публикует событие в фоне.
//
// Сбой шины не влияет на результат проверки: решение разработчика уже
// сохранено, и откатывать его из-за недоступного брокера нельзя. Цена —
// потерянное начисление при отказе Kafka; чинится outbox-таблицей, когда
// это станет критично.
//
// Контекст берётся новый, а не производный от запросного: тот отменяется
// сразу после ответа, и публикация не успевала бы уйти.
func publishAsync(logger *zap.Logger, eventType string, publish func(context.Context) error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
		defer cancel()

		if err := publish(ctx); err != nil {
			logger.Warn("failed to publish bug event",
				zap.String("event_type", eventType),
				zap.Error(err),
			)
		}
	}()
}
