package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestKeyForPathMatchesBuildKey защищает связь между записью и инвалидацией:
// Middleware кладёт ответ под ключ buildKey, а сброс ищет его через
// KeyForPath. Разойдись они — Delete удалял бы несуществующий ключ, кеш
// оставался бы жив, и баг выглядел бы как «инвалидация не работает», хотя
// вызывается она исправно.
func TestKeyForPathMatchesBuildKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	paths := []string{
		"/api/v2/profile/42/statistics/summary",
		"/api/v2/profile/1/statistics/monthly",
		"/",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, path, nil)

			want := buildKey(c, "summary_cache")
			got := KeyForPath("summary_cache", path)

			if got != want {
				t.Errorf("KeyForPath = %q, buildKey = %q", got, want)
			}
		})
	}
}

func TestKeyForPathWithoutPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)

	if got, want := KeyForPath("", "/x"), buildKey(c, ""); got != want {
		t.Errorf("KeyForPath = %q, buildKey = %q", got, want)
	}
}
