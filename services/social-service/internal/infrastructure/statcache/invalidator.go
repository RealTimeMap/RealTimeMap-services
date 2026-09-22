// Package statcache сбрасывает закешированные ответы статистики профиля.
//
// Ответы /statistics/* кешируются middleware на пять минут. Без сброса
// подписка или новый друг не отражались бы на профиле до истечения TTL:
// пользователь видит прежние числа и считает, что действие не сработало.
package statcache

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware/cache"
)

// endpoint — закешированный ответ статистики: последний сегмент пути и
// префикс, под которым middleware его сохранила.
type endpoint struct {
	path   string
	prefix string
}

// dependentEndpoints — ответы, зависящие от числа подписок, подписчиков и
// друзей.
var dependentEndpoints = []endpoint{
	{path: "summary", prefix: "summary_cache"},
	{path: "monthly", prefix: "monthly_cache"},
}

// Invalidator сбрасывает кеш статистики конкретных профилей.
type Invalidator struct {
	cache  cache.Cache
	logger *zap.Logger
}

func New(c cache.Cache, logger *zap.Logger) *Invalidator {
	return &Invalidator{cache: c, logger: logger}
}

// InvalidateProfiles сбрасывает кеш статистики каждого из профилей.
func (i *Invalidator) InvalidateProfiles(ctx context.Context, profileIDs ...uint) {
	for _, id := range profileIDs {
		for _, e := range dependentEndpoints {
			key := cache.KeyForPath(e.prefix,
				fmt.Sprintf("/api/v2/profile/%d/statistics/%s", id, e.path))

			if err := i.cache.Delete(ctx, key); err != nil {
				i.logger.Warn("failed to invalidate stat cache",
					zap.String("key", key),
					zap.Uint("profile_id", id),
					zap.Error(err))
			}
		}
	}
}
