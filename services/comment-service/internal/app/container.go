package app

import (
	producer2 "github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service/comment"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	CommentService *comment.Service
	EventPublisher service.EventPublisher

	Logger *zap.Logger
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {

	// Репозитории
	commentRepo := postgres.NewPgCommentRepository(db, logger)

	// Kafka producer (только если включен)
	var publisher service.EventPublisher
	if cfg.Kafka.Enabled {
		p := producer2.New(
			producer2.DefaultConfig().
				WithBrokers(cfg.Kafka.Brokers[0]).
				WithTopic(cfg.Kafka.ProducerTopic),
			producer2.WithLogger(logger),
		)
		publisher = kafka.NewCommentPublisher(p, logger)
		logger.Info("Kafka event publisher initialized")
	} else {
		publisher = &service.NoOpEventPublisher{}
		logger.Info("Using NoOp event publisher (Kafka disabled)")
	}

	// Сервисы
	commentService := comment.NewCommentService(commentRepo, publisher, logger)

	return &Container{
		CommentService: commentService,
		EventPublisher: publisher,

		Logger: logger,
	}
}

func (c *Container) Close() error {
	if closer, ok := c.EventPublisher.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
