package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/gamification"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	grpcserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/grpc"
	httpserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	httptransport "github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/transport/http"
	kafkatransport "github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/transport/kafka"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	db.AutoMigrate(&model.Level{}, &model.UserProgress{}, &model.Achievement{}, &model.UserAchievement{}, &model.XPReward{}, &model.EventRule{}, &model.XPOperation{}, &model.UserAchievementCount{})

	// Services
	container := app.NewContainer(cfg, db, log)

	// HTTP Server
	httpServer := httpserver.NewServer(cfg.HTTP, log)
	httpServer.Router().Static("/store", "./store")
	httptransport.RegisterRoutes(httpServer.Router(), container)

	// gRPC Server
	grpcServer, err := grpcserver.NewServer(cfg.GRPC, container.Logger, func(s *grpc.Server) {
		gamification.RegisterProgressServiceServer(s, container.ProgressGrpcHandler)
	})
	if err != nil {
		log.Fatal("failed to init gRPC server", zap.Error(err))
	}

	// Kafka Consumer
	kafkaHandler := kafkatransport.NewHandler(container.EventGamificationService, container.AchievementService, log)
	kafkaConsumer := consumer.New(
		consumer.DefaultConfig().
			WithBrokers(cfg.Kafka.Brokers...).
			WithTopics(cfg.Kafka.Topics...).
			WithGroupID(cfg.Kafka.GroupID),
		kafkaHandler.HandleMessage,
		log,
	)

	if err := runner.Run(log, httpServer, grpcServer, kafkaConsumer); err != nil {
		log.Fatal("Failed to start gamification service", zap.Error(err))
	}

	log.Info("Gamification Service stopped")
}
