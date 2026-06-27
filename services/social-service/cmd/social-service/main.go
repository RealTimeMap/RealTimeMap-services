package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	profilepb "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	grpctransport "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/grpc"
	httpserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	httptransport "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/kafka"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "social-service")
	defer log.Sync()

	log.Info("Starting social service", zap.String("env", cfg.Env))

	db := database.MustNew(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}, log)
	defer database.Close(db)

	db.AutoMigrate(&model.Profile{}, &model.Friendship{}, &model.BlockedUser{})

	container := app.NewContainer(cfg, db, log)
	defer container.Close()

	httpServer := httpserver.NewServer(cfg.Http, log)
	httpServer.Router().Static("/store", cfg.Storage.BasePath)
	httptransport.RegisterRoutes(httpServer.Router(), container)

	grpcServer, err := grpctransport.NewServer(cfg.GRPC, log, func(s *grpc.Server) {
		profilepb.RegisterProfileServiceServer(s, container.ProfileGRPCHandler)
	})
	if err != nil {
		log.Fatal("failed to init gRPC server", zap.Error(err))
	}

	kafkaHandler := kafka.NewHandler(container.ProfileService, log)
	kafkaConsumer := consumer.New(
		consumer.DefaultConfig().
			WithBrokers(cfg.Kafka.Brokers...).
			WithTopics(cfg.Kafka.Topics...).
			WithGroupID(cfg.Kafka.GroupID),
		kafkaHandler.HandleMessage,
		log,
	)

	if err := runner.Run(log, httpServer, grpcServer, kafkaConsumer); err != nil {
		log.Error("Server error", zap.Error(err))
	}

	log.Info("Social Service stopped")
}
