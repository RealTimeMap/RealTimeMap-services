package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Config — конфигурация producer.
type Config struct {
	// Brokers — адреса Kafka брокеров.
	Brokers []string

	// Topic — название топика по умолчанию.
	Topic string

	// BatchSize — количество сообщений в batch (по умолчанию 100).
	BatchSize int

	// BatchTimeout — таймаут формирования batch (по умолчанию 10ms).
	BatchTimeout time.Duration

	// Async — асинхронная отправка без подтверждения.
	Async bool
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() Config {
	return Config{
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		Async:        false,
	}
}

// Header заголовоки метаинформации для сообщения
type Header struct {
	Key   string
	Value []byte
}

// Producer отправляет события в Kafka.
type Producer struct {
	writer       *kafka.Writer
	logger       *zap.Logger
	defaultTopic string // топик по умолчанию из конфига
}

// New создаёт новый Producer.
func New(cfg Config, opts ...Option) *Producer {
	writer := &kafka.Writer{
		Addr: kafka.TCP(cfg.Brokers...),
		// Topic НЕ устанавливается в Writer - будет в Message
		Balancer:     &kafka.Hash{}, // партиционирование по ключу
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		Async:        cfg.Async,
	}

	p := &Producer{
		writer:       writer,
		logger:       zap.NewNop(),
		defaultTopic: cfg.Topic,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Option — опция конфигурации Producer.
type Option func(*Producer)

// WithLogger устанавливает кастомный logger.
func WithLogger(logger *zap.Logger) Option {
	return func(p *Producer) {
		p.logger = logger
	}
}

// WithBrokers устанавливает адреса брокеров.
func (c Config) WithBrokers(brokers ...string) Config {
	c.Brokers = brokers
	return c
}

// WithTopic устанавливает топик по умолчанию.
func (c Config) WithTopic(topic string) Config {
	c.Topic = topic
	return c
}

// Publish отправляет событие в топик по умолчанию.
// key используется для партиционирования (события с одинаковым ключом попадут в одну партицию).
func (p *Producer) Publish(ctx context.Context, key string, event any, headers []Header) error {
	return p.PublishTo(ctx, "", key, event, headers)
}

// PublishTo отправляет событие в указанный топик.
// Если topic пустой — используется топик по умолчанию из конфига.
func (p *Producer) PublishTo(ctx context.Context, topic, key string, event any, headers []Header) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Используем default топик если topic не указан
	if topic == "" {
		topic = p.defaultTopic
	}

	kafkaHeaders := make([]kafka.Header, len(headers))
	for i, h := range headers {
		kafkaHeaders[i] = kafka.Header{Key: h.Key, Value: h.Value}
	}

	msg := kafka.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   payload,
		Headers: kafkaHeaders,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Error("publish failed",
			zap.String("topic", topic),
			zap.String("key", key),
			zap.String("error", err.Error()),
		)
		return fmt.Errorf("write message: %w", err)
	}

	p.logger.Debug("event published",
		zap.String("topic", topic),
		zap.String("key", key),
	)

	return nil
}

// PublishBatch отправляет несколько событий одним batch.
func (p *Producer) PublishBatch(ctx context.Context, messages []Message) error {
	kafkaMessages := make([]kafka.Message, len(messages))
	for i, m := range messages {
		payload, err := json.Marshal(m.Event)
		if err != nil {
			return fmt.Errorf("marshal event %d: %w", i, err)
		}

		kafkaMessages[i] = kafka.Message{
			Topic: m.Topic,
			Key:   []byte(m.Key),
			Value: payload,
		}
	}

	if err := p.writer.WriteMessages(ctx, kafkaMessages...); err != nil {
		return fmt.Errorf("write batch: %w", err)
	}

	return nil
}

// Message — сообщение для batch отправки.
type Message struct {
	Topic string
	Key   string
	Event any
}

// PublishWithMeta отправляет событие с метаданными в headers.
// Использует EventMeta для стандартных полей (event_type, user_id, source_id, timestamp).
func (p *Producer) PublishWithMeta(ctx context.Context, meta EventMeta, payload any) error {
	return p.PublishToWithMeta(ctx, "", meta, payload)
}

// PublishToWithMeta отправляет событие в указанный топик с метаданными.
func (p *Producer) PublishToWithMeta(ctx context.Context, topic string, meta EventMeta, payload any) error {
	headers := meta.ToHeaders()
	return p.PublishTo(ctx, topic, meta.UserID, payload, headers)
}

// EventMeta — метаданные события для headers.
type EventMeta struct {
	EventType string
	UserID    string
	SourceID  string
	Timestamp string
}

// ToHeaders конвертирует EventMeta в slice Headers.
func (m EventMeta) ToHeaders() []Header {
	headers := make([]Header, 0, 4)

	if m.EventType != "" {
		headers = append(headers, Header{Key: "event_type", Value: []byte(m.EventType)})
	}
	if m.UserID != "" {
		headers = append(headers, Header{Key: "user_id", Value: []byte(m.UserID)})
	}
	if m.SourceID != "" {
		headers = append(headers, Header{Key: "source_id", Value: []byte(m.SourceID)})
	}
	if m.Timestamp != "" {
		headers = append(headers, Header{Key: "timestamp", Value: []byte(m.Timestamp)})
	}

	return headers
}

// Close закрывает producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
