package kafka

import "github.com/segmentio/kafka-go"

// Стандартные ключи headers
const (
	HeaderEventType = "event_type"
	HeaderUserID    = "user_id"
	HeaderSourceID  = "source_id"
	HeaderTimestamp = "timestamp"
)

// EventMeta — метаданные события из headers.
type EventMeta struct {
	EventType string
	UserID    string
	SourceID  string
	Timestamp string
}

// ExtractMeta извлекает метаданные из headers сообщения.
func ExtractMeta(msg kafka.Message) EventMeta {
	meta := EventMeta{}
	for _, h := range msg.Headers {
		switch h.Key {
		case HeaderEventType:
			meta.EventType = string(h.Value)
		case HeaderUserID:
			meta.UserID = string(h.Value)
		case HeaderSourceID:
			meta.SourceID = string(h.Value)
		case HeaderTimestamp:
			meta.Timestamp = string(h.Value)
		}
	}
	return meta
}

// GetHeader возвращает значение header по ключу.
func GetHeader(msg kafka.Message, key string) string {
	for _, h := range msg.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// MakeHeaders создаёт slice headers из EventMeta.
func MakeHeaders(meta EventMeta) []kafka.Header {
	headers := make([]kafka.Header, 0, 4)

	if meta.EventType != "" {
		headers = append(headers, kafka.Header{
			Key:   HeaderEventType,
			Value: []byte(meta.EventType),
		})
	}
	if meta.UserID != "" {
		headers = append(headers, kafka.Header{
			Key:   HeaderUserID,
			Value: []byte(meta.UserID),
		})
	}
	if meta.SourceID != "" {
		headers = append(headers, kafka.Header{
			Key:   HeaderSourceID,
			Value: []byte(meta.SourceID),
		})
	}
	if meta.Timestamp != "" {
		headers = append(headers, kafka.Header{
			Key:   HeaderTimestamp,
			Value: []byte(meta.Timestamp),
		})
	}

	return headers
}
