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
	grpctransport "github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/transport/grpc"
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

	// gRPC Server
	grpcHandler := grpctransport.NewHandler(container.ProgressRepo, container.LevelService, log)
	grpcServer, err := grpctransport.NewServer(grpcHandler, cfg.GRPC.Port, log)
	if err != nil {
		log.Fatal("Failed to create gRPC server", zap.Error(err))
	}

	// Kafka Consumer
	kafkaCfg := consumer.DefaultConfig().
		WithBrokers(cfg.Kafka.Brokers...).
		WithTopics(cfg.Kafka.Topics...).
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
		grpcServer.Stop()
		cancel()
	}()

	// Start gRPC server in goroutine
	go func() {
		if err := grpcServer.Run(); err != nil {
			log.Error("gRPC server error", zap.Error(err))
			cancel()
		}
	}()

	// Start Kafka consumer
	log.Info("Starting Kafka consumer",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.Strings("topics", cfg.Kafka.Topics),
		zap.String("group_id", cfg.Kafka.GroupID),
	)

	if err := kafkaConsumer.Run(ctx, handler.HandleMessage); err != nil {
		log.Error("Kafka consumer error", zap.Error(err))
	}

	log.Info("Gamification Service stopped")
}
