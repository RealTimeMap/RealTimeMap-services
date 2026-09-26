package main

import (
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	httpserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/userdeleted"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment/reaction"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/persistence/postgres"
	httptransport "github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "comment-service")
	defer log.Sync()

	log.Info("Starting Comment Service", zap.String("env", cfg.Env))

	// Database
	db := database.MustNew(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}, log)
	defer database.Close(db)
	db.AutoMigrate(&comment.Comment{}, &reaction.Reaction{})

	// Дизлайки удалены: убираем устаревшую колонку, AutoMigrate её сам не дропает
	if db.Migrator().HasColumn(&comment.Comment{}, "dislikes_count") {
		if err := db.Migrator().DropColumn(&comment.Comment{}, "dislikes_count"); err != nil {
			log.Error("failed to drop dislikes_count column", zap.Error(err))
		}
	}

	container := app.MustContainer(cfg, db, log)
	defer container.Close()

	httpServer := httpserver.NewServer(cfg.HTTP, log)
	httptransport.RegisterRoutes(httpServer.Router(), container)

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
		// Без консьюмера удаление аккаунта не обезличит комментарии
		log.Warn("Kafka consumer disabled: user.deleted will not be processed")
	}

	if err := runner.Run(log, servers...); err != nil {
		log.Fatal("Comment Service error", zap.Error(err))
	}

	log.Info("Comment Service stopped")
}
