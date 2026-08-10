package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	pkghttp "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	httpTransport "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/transport/http"
	kafkatransport "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/transport/kafka"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "smtp-service")
	defer log.Sync()

	db := database.MustNew(cfg.Database.ToPkg(), log)
	defer database.Close(db)

	if err := db.AutoMigrate(&email.Email{}, &email.Event{}); err != nil {
		log.Fatal("failed to run migrations", zap.Error(err))
	}

	container, err := app.NewContainer(cfg, db, log)
	if err != nil {
		log.Fatal("failed to initialize container", zap.Error(err))
	}

	httpServer := pkghttp.NewServer(cfg.HTTP, log)
	httpTransport.RegisterRoutes(httpServer.Router(), container)

	// Kafka-вход: хендлер только ставит письмо в очередь и отдаёт управление,
	// чтобы offset коммитился сразу, а не после SMTP-диалога.
	kafkaHandler := kafkatransport.NewHandler(container.Emailer, log)
	kafkaConsumer := consumer.New(
		consumer.DefaultConfig().
			WithBrokers(cfg.Kafka.Brokers...).
			WithTopics(cfg.Kafka.Topics...).
			WithGroupID(cfg.Kafka.GroupID),
		kafkaHandler.HandleMessage,
		log,
	)

	// runner.Run останавливает все компоненты одной группой по SIGINT/SIGTERM.
	if err := runner.Run(log, httpServer, container.Workers, container.Reaper, kafkaConsumer); err != nil {
		log.Fatal("failed to start smtp service", zap.Error(err))
	}
}
