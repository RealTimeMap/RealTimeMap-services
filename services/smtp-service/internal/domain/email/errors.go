package email

import (
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

// ErrInvalidEmail — адрес получателя пустой или не является валидным адресом.
// Такое письмо в очередь не попадает: занимать воркера заведомо провальной
// отправкой незачем.
func ErrInvalidEmail(value string) error {
	return apperror.NewInvalidFormatError("to", "email", value)
}

// ErrNotFound — письма с таким идентификатором нет.
func ErrNotFound(id string) error {
	return apperror.NewNotFoundErrorByID("email", id)
}

// Классы ошибок отправки. Определяют, будет ли предпринята повторная попытка.
var (
	// ErrTransient — временный отказ: сеть, таймаут, SMTP 4xx.
	// Письмо возвращается в очередь с отложенным ScheduledAt.
	ErrTransient = errors.New("transient send error")

	// ErrPermanent — отказ, который не пройдёт сам: SMTP 5xx, ошибка
	// аутентификации, битый адрес.
	//
	// Повторять 5xx не просто бесполезно, а вредно: попытки доставки на
	// несуществующие адреса портят репутацию отправителя, и провайдер
	// начинает резать весь трафик, включая валидный.
	ErrPermanent = errors.New("permanent send error")
)
