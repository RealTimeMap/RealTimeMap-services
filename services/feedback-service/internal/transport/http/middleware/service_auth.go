// Package middleware содержит фильтры HTTP-запросов сервиса.
package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ServiceKeyHeader — заголовок межсервисного ключа. Имя то же, что у
// smtp-service: сервисы платформы аутентифицируют друг друга одинаково.
const ServiceKeyHeader = "X-Api-Key"

// ServiceOnly пропускает только вызовы других сервисов платформы.
//
// Ключ статический и лежит в конфиге: полноценный реестр ключей с базой,
// как в smtp-service, здесь избыточен — потребитель ровно один
// (таск-менеджер), и заводить ради него таблицу значило бы усложнить
// сервис без выигрыша.
//
// Пустой ключ в конфиге закрывает маршруты полностью: незаполненная
// настройка не должна открывать доступ всем подряд.
func ServiceOnly(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"message": "service api is disabled",
				"error":   "service_api_key is not configured",
			})
			return
		}

		provided := c.GetHeader(ServiceKeyHeader)

		// Сравнение за постоянное время: обычное == завершается на первом
		// несовпавшем байте, и по времени ответа ключ можно подобрать.
		if subtle.ConstantTimeCompare([]byte(provided), []byte(key)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
				"error":   "valid service api key required",
			})
			return
		}

		c.Next()
	}
}
