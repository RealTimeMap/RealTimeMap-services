package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	kafkahandler "github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/transport/kafka"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "gamification-service")
	defer log.Sync()

	log.Info("Starting Gamification Service", zap.String("env", cfg.Env))

	// Database
	db := database.MustNew(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}, log)
	defer database.Close(db)
	db.AutoMigrate(&model.EventConfig{}, &model.Level{}, &model.UserProgress{}, &model.UserExpHistory{})

	// Services
	container := app.NewContainer(db, log)

	// Kafka Consumer
	kafkaCfg := consumer.DefaultConfig().
		WithBrokers(cfg.Kafka.Brokers...).
		WithTopic(cfg.Kafka.Topic).
		WithGroupID(cfg.Kafka.GroupID)

	kafkaConsumer := consumer.New(kafkaCfg, log)
	defer kafkaConsumer.Close()

	handler := kafkahandler.NewHandler(container.GamificationService, log)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Info("Shutdown signal received")
		cancel()
	}()

	// Start consuming
	log.Info("Starting Kafka consumer",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.String("topic", cfg.Kafka.Topic),
		zap.String("group_id", cfg.Kafka.GroupID),
	)

	if err := kafkaConsumer.Run(ctx, handler.HandleMessage); err != nil {
		log.Error("Kafka consumer error", zap.Error(err))
	}

	log.Info("Gamification Service stopped")
}
