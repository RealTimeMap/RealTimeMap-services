package subscription

import "context"

// SubscriptionCreated — состоявшаяся подписка, уходящая в шину.
type SubscriptionCreated struct {
	SubscriberID uint
	TargetID     uint

	// SubscriberName — имя подписавшегося для текста уведомления. Пустое,
	// если профиль не удалось прочитать: событие всё равно осмысленно,
	// потребитель покажет «У вас новый подписчик».
	SubscriberName string
}

// EventPublisher — порт публикации событий подписки.
//
// Интерфейс объявлен здесь, у потребителя: доменный слой не должен знать про
// Kafka, а реализация живёт в infrastructure/kafka.
type EventPublisher interface {
	PublishSubscriptionCreated(ctx context.Context, e SubscriptionCreated) error
}

// NoOpEventPublisher — заглушка на случай выключенной шины.
//
// Подписка важнее события: без брокера она всё так же оформляется, теряется
// только push-уведомление адресату.
type NoOpEventPublisher struct{}

func (NoOpEventPublisher) PublishSubscriptionCreated(context.Context, SubscriptionCreated) error {
	return nil
}
