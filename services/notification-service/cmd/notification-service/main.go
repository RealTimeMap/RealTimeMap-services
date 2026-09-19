package main

import (
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	httpserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
	httptransport "github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/transport/http"
	kafkatransport "github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/transport/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/worker"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "notification-service")
	defer log.Sync()

	log.Info("Starting Notification Service", zap.String("env", cfg.Env))

	db := database.MustNew(cfg.Database.ToPkg(), log)
	defer database.Close(db)

	if err := db.AutoMigrate(&token.Model{}); err != nil {
		log.Fatal("failed to migrate", zap.Error(err))
	}

	container := app.NewContainer(cfg, db, log)

	httpServer := httpserver.NewServer(cfg.HTTP, log)
	httptransport.RegisterRoutes(httpServer.Router(), container)

	kafkaHandler := kafkatransport.NewHandler(container.NotifyUseCase.NotifyEvent, log)
	kafkaConsumer := consumer.New(
		consumer.DefaultConfig().
			WithBrokers(cfg.Kafka.Brokers...).
			WithTopics(cfg.Kafka.Topics...).
			WithGroupID(cfg.Kafka.GroupID),
		kafkaHandler.HandleMessage,
		log,
	)

	summaryWorker := worker.NewSummary(
		container.CollapseLimiter,
		container.NotifyUseCase.NotifyEvent,
		cfg.Collapse.SummaryInterval,
		cfg.Collapse.SummaryBatch,
		log,
	)

	if err := runner.Run(log, httpServer, kafkaConsumer, summaryWorker); err != nil {
		log.Fatal("Notification Service error", zap.Error(err))
	}

	log.Info("Notification Service stopped")
}
