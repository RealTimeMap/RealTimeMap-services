package validation

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DomainErrorMapper маппит доменные ошибки в ValidationError
type DomainErrorMapper struct {
	mappings map[error]ValidationError
}

// NewDomainErrorMapper создает новый маппер доменных ошибок
func NewDomainErrorMapper() *DomainErrorMapper {
	return &DomainErrorMapper{
		mappings: make(map[error]ValidationError),
	}
}

// Register регистрирует маппинг доменной ошибки в ValidationError
func (m *DomainErrorMapper) Register(domainErr error, validationErr ValidationError) *DomainErrorMapper {
	m.mappings[domainErr] = validationErr
	return m
}

// MapAndAbort проверяет ошибку и возвращает соответствующий HTTP статус
// Если ошибка зарегистрирована в маппере - возвращает 422 с ValidationError
// Если нет - возвращает 500 Internal Server Error
func (m *DomainErrorMapper) MapAndAbort(c *gin.Context, err error) {
	// Проверяем, есть ли прямое совпадение
	if validationErr, ok := m.mappings[err]; ok {
		Abort(c, validationErr)
		return
	}

	// Проверяем через domainerrors.Is (для wrapped domainerrors)
	for domainErr, validationErr := range m.mappings {
		if errors.Is(err, domainErr) {
			Abort(c, validationErr)
			return
		}
	}

	// Если ошибка не распознана - возвращаем 500
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
	})
}
