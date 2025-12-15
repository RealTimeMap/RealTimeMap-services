package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Handler[T any] interface {
	Handle(ctx context.Context, event T) error
}

type HandlerFunc[T any] func(ctx context.Context, event T) error

func (f HandlerFunc[T]) Handle(ctx context.Context, event T) error {
	return f(ctx, event)
}

type Router[T any] struct {
	handlers    map[string]Handler[T]
	typeExtract func(T) string
}

func NewRouter[T any](typeExtract func(T) string) *Router[T] {
	return &Router[T]{
		handlers:    make(map[string]Handler[T]),
		typeExtract: typeExtract,
	}
}

// Register регистрирует обработчик для типа события.
func (r *Router[T]) Register(eventType string, handler Handler[T]) *Router[T] {
	r.handlers[eventType] = handler
	return r
}

// RegisterFunc регистрирует функцию-обработчик для типа события.
func (r *Router[T]) RegisterFunc(eventType string, handler HandlerFunc[T]) *Router[T] {
	r.handlers[eventType] = handler
	return r
}

// Route направляет событие соответствующему обработчику.
func (r *Router[T]) Route(ctx context.Context, event T) error {
	eventType := r.typeExtract(event)

	handler, exists := r.handlers[eventType]
	if !exists {
		// Событие не зарегистрировано — пропускаем
		return nil
	}

	return handler.Handle(ctx, event)
}

// MessageHandler возвращает функцию для использования с Consumer.Run.
// Парсит JSON и маршрутизирует событие.
func (r *Router[T]) MessageHandler() MessageHandler {
	return func(ctx context.Context, msg kafka.Message) error {
		var event T
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return Skip(fmt.Errorf("unmarshal event: %w", err))
		}

		return r.Route(ctx, event)
	}
}

// HasHandler проверяет, зарегистрирован ли обработчик для типа события.
func (r *Router[T]) HasHandler(eventType string) bool {
	_, exists := r.handlers[eventType]
	return exists
}

// RegisteredTypes возвращает список зарегистрированных типов событий.
func (r *Router[T]) RegisteredTypes() []string {
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}
