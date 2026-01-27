package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service/comment"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	CommentService *comment.Service
	Producer       *producer.Producer

	Logger *zap.Logger
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {

	// Репозитории
	commentRepo := postgres.NewPgCommentRepository(db, logger)

	// Kafka producer (только если включен)
	var p *producer.Producer
	if cfg.Kafka.Enabled {
		p = producer.New(
			producer.DefaultConfig().WithBrokers(cfg.Kafka.Brokers[0]).WithTopic(cfg.Kafka.ProducerTopic),
			producer.WithLogger(logger),
		)
		logger.Info("Kafka producer initialized", zap.String("topic", cfg.Kafka.ProducerTopic))
	} else {
		logger.Info("Kafka producer disabled")
	}

	// Сервисы
	commentService := comment.NewCommentService(commentRepo, p, logger)

	return &Container{
		CommentService: commentService,
		Producer:       p,

		Logger: logger,
	}
}
