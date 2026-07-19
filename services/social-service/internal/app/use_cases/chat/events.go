package chat

import "context"

// ChatEventType — тип realtime-события чата, доставляемого клиенту через сокет.
type ChatEventType string

const (
	// EventMessageNew — новое сообщение в чате. Payload = MessageResult.
	EventMessageNew ChatEventType = "message.new"
)

// ChatEvent — доменное событие, которое useCase публикует после изменения
// состояния чата. RecipientIDs — кому доставить (обычно участники чата, кроме
// инициатора). Payload — уже готовый DTO ответа (тот же, что уходит по HTTP).
type ChatEvent struct {
	Type         ChatEventType
	RecipientIDs []uint
	Payload      any
}

// EventPublisher — порт доставки realtime-событий. useCase зависит только от него
// и ничего не знает про Socket.IO/Redis. Реализация (socketpub) шлёт событие в
// комнаты user:<id>; межинстансную рассылку берёт на себя Redis adapter.
type EventPublisher interface {
	Publish(ctx context.Context, e ChatEvent) error
}
