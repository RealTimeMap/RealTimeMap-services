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

// Producer отправляет события в Kafka.
type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

// New создаёт новый Producer.
func New(cfg Config, opts ...Option) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.Hash{}, // партиционирование по ключу
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		Async:        cfg.Async,
	}

	p := &Producer{
		writer: writer,
		logger: zap.NewNop(),
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
func (p *Producer) Publish(ctx context.Context, key string, event any) error {
	return p.PublishTo(ctx, "", key, event)
}

// PublishTo отправляет событие в указанный топик.
// Если topic пустой — используется топик по умолчанию из конфига.
func (p *Producer) PublishTo(ctx context.Context, topic, key string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: payload,
	}

	if topic != "" {
		msg.Topic = topic
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

// Close закрывает producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
