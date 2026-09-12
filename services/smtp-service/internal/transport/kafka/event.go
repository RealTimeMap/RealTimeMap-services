package kafka

import (
	"encoding/json"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
)

// Типы событий, на которые сервис отправляет письма.
const (
	EventUserRegistered = events.UserRegistered
	EventCommentCreated = events.CommentCreated
)

// UserRegistered — событие регистрации пользователя из auth-сервиса.
//
// Тип объявлен здесь, а не взят из pkg/events напрямую, потому что несёт
// разбор двух форматов сразу (см. decodeUserRegistered).
type UserRegistered = events.UserRegisteredPayload

// decodeUserRegistered достаёт payload регистрации из тела сообщения.
//
// Понимает два формата. Новый — конверт {type, payload}, общий с остальными
// сервисами. Старый — плоский объект, который auth-сервис слал до перехода на
// конверт: такие сообщения ещё лежат в топике по retention, и отказ их читать
// означал бы потерю писем за период до деплоя.
//
// Различаются по наличию ключа payload: в плоском формате его нет.
func decodeUserRegistered(body []byte) (UserRegistered, error) {
	var envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return UserRegistered{}, fmt.Errorf("unmarshal envelope: %w", err)
	}

	raw := envelope.Payload
	if len(raw) == 0 {
		raw = body
	}

	var payload UserRegistered
	if err := json.Unmarshal(raw, &payload); err != nil {
		return UserRegistered{}, fmt.Errorf("unmarshal user.registered payload: %w", err)
	}
	return payload, nil
}

// decodeComment достаёт payload комментария. Формат только конвертный:
// comment-service с самого начала публикует в нём.
func decodeComment(body []byte) (events.CommentPayload, error) {
	var event events.CommentEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return events.CommentPayload{}, fmt.Errorf("unmarshal comment event: %w", err)
	}
	return event.Payload, nil
}
