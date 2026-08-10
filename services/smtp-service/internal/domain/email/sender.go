package email

import "context"

// OutgoingMessage — готовое к отправке письмо. Тело уже отрендерено, шаблоны
// на этом уровне не фигурируют: отправляющая сторона про них не знает.
type OutgoingMessage struct {
	To      string
	Subject string
	HTML    string
}

// Sender отправляет письмо через почтового провайдера.
//
// Шов для смены провайдера: MVP работает через SMTP Yandex, переезд на
// SES/SendGrid не затрагивает ни очередь, ни воркеров.
//
// Реализация обязана классифицировать ошибки, оборачивая их в ErrTransient
// или ErrPermanent — от этого зависит, будет ли повторная попытка.
type Sender interface {
	Send(ctx context.Context, msg OutgoingMessage) error
}

// Recipient — получатель письма.
type Recipient struct {
	Email    string
	Username string
}

// RecipientResolver находит контактные данные пользователя.
//
// Шов, за которым скрыт источник email. В MVP адрес приезжает прямо в событии
// (auth-service ещё не написан: proto/user/service.proto существует, но
// реализации, сгенерированного кода и клиента нет). Когда UserService
// появится, реализация меняется на gRPC-вызов, и продюсеры перестают
// рассылать персональные данные через Kafka.
type RecipientResolver interface {
	Resolve(ctx context.Context, userID uint64) (Recipient, error)
}
