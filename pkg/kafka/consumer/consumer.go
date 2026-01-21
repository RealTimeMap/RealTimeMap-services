package consumer

import (
	"context"
	"errors"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger/sl"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type MessageHandler func(ctx context.Context, message kafka.Message) error

type Consumer struct {
	reader     *kafka.Reader
	log        *zap.Logger
	retryDelay time.Duration
}

func New(cfg Config, log *zap.Logger) *Consumer {
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
	return &Consumer{
		reader:     reader,
		log:        log,
		retryDelay: time.Second,
	}
}

func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	cfg := c.reader.Config()
	topics := cfg.GroupTopics
	if len(topics) == 0 && cfg.Topic != "" {
		topics = []string{cfg.Topic}
	}
	c.log.Info("starting consumer", zap.Strings("topics", topics), sl.String("groupId", cfg.GroupID))

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				c.log.Info("Context canceled. Stop consumer.")
				return c.reader.Close()
			}
			c.log.Error("Error reading message", zap.Error(err))
			continue
		}
		c.processMessage(ctx, msg, handler)
	}
}

func (c *Consumer) processMessage(ctx context.Context, message kafka.Message, handler MessageHandler) {
	logger := c.log.With(
		zap.String("topic", message.Topic),
		zap.Int("partition", message.Partition),
		zap.Int64("time", message.Offset))

	err := handler(ctx, message)

	switch {
	case err == nil:
		if commitErr := c.reader.CommitMessages(ctx, message); commitErr != nil {
			logger.Error("Error committing message", zap.Error(commitErr))
		}
	case errors.Is(err, ErrSkip):
		logger.Warn("Skipping message", zap.Error(err))
		if commitErr := c.reader.CommitMessages(ctx, message); commitErr != nil {
			logger.Error("Error committing message", zap.Error(commitErr))
		}
	case errors.Is(err, ErrRetryable):
		logger.Warn("Retryable error, message will be reprocessed",
			zap.String("duration", c.retryDelay.String()),
			zap.Error(err),
		)
		time.Sleep(c.retryDelay)
	case errors.Is(err, ErrFatal):
		logger.Error("Fatal error, message will be reprocessed", zap.Error(err))
		c.reader.CommitMessages(ctx, message)
	default:
		logger.Error("Error reading message", zap.Error(err))
		_ = c.reader.CommitMessages(ctx, message)
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
