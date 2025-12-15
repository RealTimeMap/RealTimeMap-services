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
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		MaxWait:        cfg.MaxWait,
		MaxBytes:       cfg.MaxBytes,
		MinBytes:       cfg.MinBytes,
		CommitInterval: cfg.CommitInterval,
	})
	return &Consumer{
		reader:     reader,
		log:        log,
		retryDelay: time.Second,
	}
}

func (c *Consumer) Run(ctx context.Context, handler MessageHandler) error {
	c.log.Info("starting consumer", sl.String("topic", c.reader.Config().Topic), sl.String("groupId", c.reader.Config().GroupID))

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
