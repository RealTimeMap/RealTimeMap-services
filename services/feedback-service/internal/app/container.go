package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/producer"
	bugcases "github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app/use_cases/bug"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	feedbackkafka "github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/infrastructure/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	BugCases *bugcases.Application
	Logger   *zap.Logger
	DB       *gorm.DB

	// ServiceApiKey — ключ межсервисных маршрутов. Транспорт берёт его
	// отсюда, чтобы не тянуть весь конфиг в слой маршрутов.
	ServiceApiKey string

	publisher bugcases.EventPublisher
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) (*Container, error) {
	bugRepo := postgres.NewPgBugRepository(db, logger)
	bugService := bug.NewService(bugRepo, logger)
	publisher := newPublisher(cfg.Kafka, logger)
	bugUseCases := &bugcases.Application{
		Create: bugcases.NewCreatorBugHandler(bugService, logger),
		List:   bugcases.NewListBugHandler(bugService, logger),
		Get:    bugcases.NewGetBugHandler(bugService, logger),
		Link:   bugcases.NewLinkBugHandler(bugService, logger),
		Review: bugcases.NewReviewBugHandler(bugService, publisher, logger),
	}

	return &Container{
		BugCases:      bugUseCases,
		Logger:        logger,
		DB:            db,
		ServiceApiKey: cfg.ServiceApiKey,
		publisher:     publisher,
	}, nil
}

// newPublisher поднимает публикацию событий, если шина включена.
//
// Включённая шина без адресов — ошибка конфигурации, но ронять сервис из-за
// неё нельзя: приём отчётов важнее наград. Падаем на заглушку и говорим об
// этом громко.
func newPublisher(cfg config.Kafka, logger *zap.Logger) bugcases.EventPublisher {
	switch {
	case !cfg.Enabled:
		logger.Info("Using NoOp event publisher (Kafka disabled)")
		return bugcases.NoOpEventPublisher{}
	case len(cfg.Brokers) == 0:
		logger.Error("Kafka enabled but no brokers configured, falling back to NoOp publisher")
		return bugcases.NoOpEventPublisher{}
	}

	p := producer.New(
		producer.DefaultConfig().
			WithBrokers(cfg.Brokers...).
			WithTopic(cfg.ProducerTopic),
		producer.WithLogger(logger),
	)
	logger.Info("Kafka event publisher initialized",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.ProducerTopic),
	)
	return feedbackkafka.NewBugPublisher(p, logger)
}

// Close закрывает соединение с шиной.
func (c *Container) Close() error {
	if closer, ok := c.publisher.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
