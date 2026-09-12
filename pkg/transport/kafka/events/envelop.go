// Package events описывает формат доменных событий, которыми обмениваются
// сервисы через Kafka.
//
// Формат один на все сервисы, включая auth на Python:
//
//	{"id": "...", "type": "comment.created", "timestamp": "...", "payload": {...}}
//
// Та же мета дублируется в headers сообщения (event_type, user_id, source_id,
// timestamp). Дублирование намеренное: headers позволяют отфильтровать или
// смаршрутизировать событие, не разбирая тело, а тело остаётся
// самодостаточным — сообщение, вычитанное из топика напрямую (kafka-ui,
// дамп для разбора инцидента), не теряет смысла без заголовков.
//
// Потребитель читает тело и падает обратно на headers, а не наоборот:
// заголовки может потерять промежуточный компонент (mirror-maker, прокси),
// тело — нет.
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Envelop — общая часть любого события: идентификатор, тип и время.
//
// ID нужен потребителям для дедупликации: Kafka гарантирует at-least-once, и
// одно и то же событие может приехать дважды после ребаланса группы.
type Envelop struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

func NewEnvelop(eventType string) Envelop {
	return Envelop{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
	}
}

// EventType возвращает тип события. Метод, а не прямое обращение к полю:
// хендлеры принимают конверт по интерфейсу, не зная конкретного типа события.
func (e Envelop) EventType() string {
	return e.Type
}

// RawEvent — конверт с неразобранным payload.
//
// Хендлер сначала читает тип, а разбирает payload уже зная, что перед ним;
// сообщение чужого типа не должно проваливать разбор.
type RawEvent struct {
	Envelop
	Payload json.RawMessage `json:"payload"`
}
