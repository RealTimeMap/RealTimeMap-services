package statcache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware/cache"
)

// TestInvalidateHitsCachedKeys — главная проверка: ключ, который удаляет
// инвалидатор, должен совпасть с тем, под которым middleware реально сохранила
// ответ. Расхождение не сломает ни сборку, ни тесты сервисов — Delete просто
// удалит несуществующий ключ, а пользователь продолжит видеть старые числа.
// Поэтому тест гоняет обе стороны: сначала настоящую middleware, потом сброс.
func TestInvalidateHitsCachedKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const profileID = 42
	paths := []string{
		"/api/v2/profile/42/statistics/summary",
		"/api/v2/profile/42/statistics/monthly",
	}
	prefixes := map[string]string{
		paths[0]: "summary_cache",
		paths[1]: "monthly_cache",
	}

	c := cache.NewMemoryCache()

	// Прогреваем кеш через ту же middleware, что работает в проде.
	for _, path := range paths {
		router := gin.New()
		router.GET(path, cache.Middleware(c, cache.Options{
			TTL:    5 * time.Minute,
			Prefix: prefixes[path],
		}), func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"subscriptionsCount": "2"})
		})

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if got := rec.Header().Get("X-Cache-Status"); got != "MISS" {
			t.Fatalf("прогрев %s: X-Cache-Status = %q, want MISS", path, got)
		}
	}

	// Убеждаемся, что кеш действительно заполнен: иначе тест прошёл бы и при
	// сломанном сбросе.
	for _, path := range paths {
		key := cache.KeyForPath(prefixes[path], path)
		if _, ok := c.Get(context.Background(), key); !ok {
			t.Fatalf("после прогрева ключ %q пуст", key)
		}
	}

	New(c, zap.NewNop()).InvalidateProfiles(context.Background(), profileID)

	for _, path := range paths {
		key := cache.KeyForPath(prefixes[path], path)
		if _, ok := c.Get(context.Background(), key); ok {
			t.Errorf("ключ %q остался в кеше после сброса", key)
		}
	}
}

// TestInvalidateOnlyTargetProfiles — сброс не должен задевать чужие профили:
// иначе каждая подписка обнуляла бы кеш посторонним.
func TestInvalidateOnlyTargetProfiles(t *testing.T) {
	c := cache.NewMemoryCache()
	ctx := context.Background()

	other := cache.KeyForPath("summary_cache", "/api/v2/profile/99/statistics/summary")
	if err := c.Set(ctx, other, []byte("x"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	New(c, zap.NewNop()).InvalidateProfiles(ctx, 42)

	if _, ok := c.Get(ctx, other); !ok {
		t.Error("сброс профиля 42 удалил кеш профиля 99")
	}
}

// TestInvalidateSurvivesCacheFailure — Redis недоступен, но действие уже
// совершено: инвалидатор обязан промолчать, а не уронить вызывающего.
func TestInvalidateSurvivesCacheFailure(t *testing.T) {
	New(failingCache{}, zap.NewNop()).InvalidateProfiles(context.Background(), 1, 2)
}

type failingCache struct{}

func (failingCache) Get(context.Context, string) ([]byte, bool) { return nil, false }
func (failingCache) Set(context.Context, string, []byte, time.Duration) error {
	return context.DeadlineExceeded
}
func (failingCache) Delete(context.Context, string) error { return context.DeadlineExceeded }
