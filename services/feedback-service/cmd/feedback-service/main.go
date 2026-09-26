package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/userdeleted"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/infrastructure/persistence/postgres"
	httpTransport "github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/transport/http"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "feedback-service")
	defer log.Sync()

	db := database.MustNew(cfg.Database, log)
	defer database.Close(db)

	db.AutoMigrate(&bug.Model{})

	container, err := app.NewContainer(cfg, db, log)
	if err != nil {
		log.Fatal("Failed to initialize container", zap.Error(err))
	}
	defer container.Close()

	httpServer := http.NewServer(cfg.Http, log)
	httpTransport.RegisterRoutes(httpServer.Router(), container)

	servers := []runner.Server{httpServer}
	if cfg.Kafka.Enabled && len(cfg.Kafka.Brokers) > 0 && len(cfg.Kafka.Topics) > 0 {
		servers = append(servers, consumer.New(
			consumer.DefaultConfig().
				WithBrokers(cfg.Kafka.Brokers...).
				WithTopics(cfg.Kafka.Topics...).
				WithGroupID(cfg.Kafka.GroupID),
			userdeleted.Handler(postgres.NewPgAccountRepository(db), log),
			log,
		))
	} else {
		// Без консьюмера удаление аккаунта не обезличит отчёты — это
		// нарушение, а не штатный режим, поэтому громко.
		log.Warn("Kafka consumer disabled: user.deleted will not be processed")
	}

	if err := runner.Run(log, servers...); err != nil {
		log.Fatal("Failed to start feedback service", zap.Error(err))
	}

}
