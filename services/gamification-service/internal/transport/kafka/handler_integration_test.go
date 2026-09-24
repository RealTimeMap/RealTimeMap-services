package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	segmentio "github.com/segmentio/kafka-go"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
)

// broker указывает на тот же брокер, что и config.yaml сервиса.
const broker = "127.0.0.1:9093"

// TestEventRoundTrip проверяет сквозной путь события: публикация настоящим
// producer'ом, чтение настоящим consumer'ом, разбор настоящим extractMeta.
//
// Тест нужен потому, что мета события собирается в двух несвязанных местах:
// издатель кладёт её в headers и в тело, а gamification-service достаёт
// оттуда по своим правилам (тело приоритетнее, userId ищется по известным
// именам полей). Расхождение здесь не ловится ни компилятором, ни unit-тестом
// хендлера — событие просто молча не засчитывается.
func TestEventRoundTrip(t *testing.T) {
	requireKafka(t)

	tests := []struct {
		name      string
		topic     string
		eventType string
		meta      producer.EventMeta
		payload   any

		wantUserID uint
	}{
		{
			name:      "markCreated",
			topic:     "mark_action-service.events",
			eventType: events.MarkCreated,
			meta: producer.EventMeta{
				EventType: "mark.created",
				UserID:    "1001",
				SourceID:  "5001",
			},
			payload:    events.NewMarkCreate(markPayload(5001, 1001)),
			wantUserID: 1001,
		},
		{
			// Ночная метка: тот же издатель, другой тип. Проверяем, что новый
			// тип доезжает и опознаётся так же, как markCreated.
			name:      "markCreatedAtNight",
			topic:     "mark_action-service.events",
			eventType: events.MarkCreatedAtNight,
			meta: producer.EventMeta{
				EventType: events.MarkCreatedAtNight,
				UserID:    "1002",
				SourceID:  "5002",
			},
			payload:    events.NewMarkTimeOfDayEvent(events.MarkCreatedAtNight, markPayload(5002, 1002)),
			wantUserID: 1002,
		},
		{
			name:      "markCreatedAtEarlyMorning",
			topic:     "mark_action-service.events",
			eventType: events.MarkCreatedAtEarlyMorning,
			meta: producer.EventMeta{
				EventType: events.MarkCreatedAtEarlyMorning,
				UserID:    "1003",
				SourceID:  "5003",
			},
			payload:    events.NewMarkTimeOfDayEvent(events.MarkCreatedAtEarlyMorning, markPayload(5003, 1003)),
			wantUserID: 1003,
		},
		{
			// Комментарий — второй издатель со своим форматом payload:
			// userId лежит в теле, и extractMeta обязан взять его оттуда.
			name:      "comment.created",
			topic:     "comment-service.events",
			eventType: events.CommentCreated,
			meta: producer.EventMeta{
				EventType: events.CommentCreated,
				UserID:    "1004",
			},
			payload: events.RawEvent{
				Envelop: events.NewEnvelop(events.CommentCreated),
				Payload: mustJSON(map[string]any{"userId": 1004, "commentId": 77}),
			},
			wantUserID: 1004,
		},
		{
			// Подтверждённый баг: bugId в теле — источник события.
			name:      "bug.confirmed",
			topic:     "feedback-service.events",
			eventType: events.BugConfirmed,
			meta: producer.EventMeta{
				EventType: events.BugConfirmed,
				UserID:    "1005",
				SourceID:  "9005",
			},
			payload:    events.NewBugConfirmed(events.BugPayload{BugID: 9005, UserID: 1005, Tag: "ui"}),
			wantUserID: 1005,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := tt.meta
			meta.Timestamp = time.Now().Format(time.RFC3339)

			from := tailOffset(t, tt.topic)
			publish(t, tt.topic, meta, tt.payload)

			// Ищем по userID: он уникален для каждого кейса и не пересекается
			// с реальными данными в топике. SourceID для маркера не годится —
			// для части событий он берётся из тела (payload.commentId), и
			// заголовок его не переопределяет.
			msg := readFrom(t, tt.topic, from, func(m segmentio.Message) bool {
				got, err := extractMeta(m)
				return err == nil && got.UserID == tt.wantUserID
			})

			got, err := extractMeta(msg)
			if err != nil {
				t.Fatalf("extractMeta: %v", err)
			}

			if got.EventType != tt.eventType {
				t.Errorf("EventType = %q, want %q", got.EventType, tt.eventType)
			}
			if got.UserID != tt.wantUserID {
				t.Errorf("UserID = %d, want %d", got.UserID, tt.wantUserID)
			}
			if got.SourceID == nil {
				t.Error("SourceID потерян")
			}
		})
	}
}

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}

func markPayload(markID, ownerID int) events.MarkPayload {
	return events.NewMarkPayload(markID, 1, ownerID, "integration test", nil)
}

func requireKafka(t *testing.T) {
	t.Helper()

	conn, err := segmentio.DialContext(context.Background(), "tcp", broker)
	if err != nil {
		t.Skipf("kafka недоступна на %s: %v", broker, err)
	}
	conn.Close()
}

func publish(t *testing.T, topic string, meta producer.EventMeta, payload any) {
	t.Helper()

	cfg := producer.DefaultConfig()
	cfg.Brokers = []string{broker}
	cfg.Topic = topic
	cfg.BatchTimeout = 10 * time.Millisecond

	p := producer.New(cfg)
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := p.PublishToWithMeta(ctx, topic, meta, payload); err != nil {
		t.Fatalf("публикация в %s: %v", topic, err)
	}
}

// tailOffset возвращает offset конца партиции — позицию, с которой появятся
// сообщения, опубликованные после вызова.
//
// Снимается ДО публикации: иначе читатель, стартующий с конца, пропустит
// собственное сообщение, а стартующий с начала вычитает всю историю топика.
func tailOffset(t *testing.T, topic string) int64 {
	t.Helper()

	conn, err := segmentio.DialLeader(context.Background(), "tcp", broker, topic, 0)
	if err != nil {
		t.Fatalf("подключение к %s: %v", topic, err)
	}
	defer conn.Close()

	_, last, err := conn.ReadOffsets()
	if err != nil {
		t.Fatalf("чтение offset %s: %v", topic, err)
	}
	return last
}

// readFrom читает топик с указанного offset, пока не встретит сообщение, для
// которого match вернёт true.
//
// Напрямую через kafka-go, а не через consumer.Consumer: тот коммитит
// offset'ы в группе, из-за чего повторный прогон не увидел бы собственных
// сообщений.
func readFrom(t *testing.T, topic string, offset int64, match func(segmentio.Message) bool) segmentio.Message {
	t.Helper()

	r := segmentio.NewReader(segmentio.ReaderConfig{
		Brokers:   []string{broker},
		Topic:     topic,
		Partition: 0,
		MaxWait:   200 * time.Millisecond,
	})
	defer r.Close()

	if err := r.SetOffset(offset); err != nil {
		t.Fatalf("SetOffset(%d): %v", offset, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			t.Fatalf("сообщение не пришло за отведённое время: %v", err)
		}
		if match(msg) {
			return msg
		}
	}
}
