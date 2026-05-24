package consumer

import (
	"context"
	"errors"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger/sl"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// MessageHandler обработчик одного сообщения Kafka.
type MessageHandler func(ctx context.Context, message kafka.Message) error

// Consumer читает сообщения из Kafka и передаёт их в handler.
// Реализует интерфейс runner.Server: Run() error / Shutdown(ctx) error.
type Consumer struct {
	reader     *kafka.Reader
	handler    MessageHandler
	log        *zap.Logger
	retryDelay time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New создаёт Consumer с handler-ом из конструктора.
// handler не может быть nil.
func New(cfg Config, handler MessageHandler, log *zap.Logger) *Consumer {
	readerCfg := kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		MaxWait:        cfg.MaxWait,
		MaxBytes:       cfg.MaxBytes,
		MinBytes:       cfg.MinBytes,
		CommitInterval: cfg.CommitInterval,
	}

	// Поддержка нескольких топиков
	if len(cfg.Topics) > 1 {
		readerCfg.GroupTopics = cfg.Topics
	} else if len(cfg.Topics) == 1 {
		readerCfg.Topic = cfg.Topics[0]
	}

	reader := kafka.NewReader(readerCfg)
	ctx, cancel := context.WithCancel(context.Background())
	return &Consumer{
		reader:     reader,
		handler:    handler,
		log:        log,
		retryDelay: time.Second,
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
	}
}

// Run блокируется до отмены внутреннего ctx (через Shutdown) или фатальной ошибки.
// Возвращает nil при штатной остановке через Shutdown.
func (c *Consumer) Run() error {
	defer close(c.done)

	cfg := c.reader.Config()
	topics := cfg.GroupTopics
	if len(topics) == 0 && cfg.Topic != "" {
		topics = []string{cfg.Topic}
	}
	c.log.Info("kafka consumer starting", zap.Strings("topics", topics), sl.String("groupId", cfg.GroupID))

	for {
		msg, err := c.reader.FetchMessage(c.ctx)
		if err != nil {
			if c.ctx.Err() != nil {
				c.log.Info("kafka consumer stopped")
				return nil
			}
			c.log.Error("error reading message", zap.Error(err))
			continue
		}
		c.processMessage(c.ctx, msg)
	}
}

// Shutdown сигналит Run завершиться и закрывает reader.
// Если ctx истечёт раньше, чем Run выйдет — возвращает ctx.Err().
func (c *Consumer) Shutdown(ctx context.Context) error {
	c.log.Info("kafka consumer stopping")
	c.cancel()
	if err := c.reader.Close(); err != nil {
		c.log.Warn("kafka reader close failed", zap.Error(err))
	}
	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Consumer) processMessage(ctx context.Context, message kafka.Message) {
	logger := c.log.With(
		zap.String("topic", message.Topic),
		zap.Int("partition", message.Partition),
		zap.Int64("offset", message.Offset))

	err := c.handler(ctx, message)

	switch {
	case err == nil:
		if commitErr := c.reader.CommitMessages(ctx, message); commitErr != nil {
			logger.Error("error committing message", zap.Error(commitErr))
		}
	case errors.Is(err, ErrSkip):
		logger.Warn("skipping message", zap.Error(err))
		if commitErr := c.reader.CommitMessages(ctx, message); commitErr != nil {
			logger.Error("error committing message", zap.Error(commitErr))
		}
	case errors.Is(err, ErrRetryable):
		logger.Warn("retryable error, message will be reprocessed",
			zap.String("duration", c.retryDelay.String()),
			zap.Error(err),
		)
		time.Sleep(c.retryDelay)
	case errors.Is(err, ErrFatal):
		logger.Error("fatal error, message committed", zap.Error(err))
		_ = c.reader.CommitMessages(ctx, message)
	default:
		logger.Error("error handling message", zap.Error(err))
		_ = c.reader.CommitMessages(ctx, message)
	}
}
