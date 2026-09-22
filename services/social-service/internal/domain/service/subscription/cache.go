package subscription

import "context"

// StatInvalidator сбрасывает кеш статистики профилей, чьи счётчики изменились.
//
// Порт объявлен здесь, у потребителя: доменный слой не должен знать про Redis
// и про HTTP-кеш, а реализация живёт в infrastructure/statcache.
type StatInvalidator interface {
	InvalidateProfiles(ctx context.Context, profileIDs ...uint)
}

// NoOpStatInvalidator — заглушка на случай выключенного кеша.
//
// Подписка важнее сброса: без Redis она всё так же оформляется, а устаревшие
// числа сами исчезнут по TTL.
type NoOpStatInvalidator struct{}

func (NoOpStatInvalidator) InvalidateProfiles(context.Context, ...uint) {}
