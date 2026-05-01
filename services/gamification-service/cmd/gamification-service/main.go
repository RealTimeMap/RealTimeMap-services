package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/proto/gamification"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	grpcserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/grpc"
	httpserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	httptransport "github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/transport/http"
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
	db.AutoMigrate(&model.EventConfig{}, &model.Level{}, &model.UserProgress{}, &model.UserExpHistory{})

	// Services
	container := app.NewContainer(cfg, db, log)

	// HTTP Server
	httpServer := httpserver.NewServer(cfg.HTTP, log)
	httptransport.RegisterRoutes(httpServer.Router(), container)

	// gRPC Server
	grpcServer, err := grpcserver.NewServer(cfg.GRPC, container.Logger, func(s *grpc.Server) {
		gamification.RegisterProgressServiceServer(s, container.ProgressGrpcHandler)
	})
	if err != nil {
		log.Fatal("Failed to start gamification service", zap.Error(err))
	}

	//// Kafka Consumer
	//kafkaCfg := consumer.DefaultConfig().
	//	WithBrokers(cfg.Kafka.Brokers...).
	//	WithTopics(cfg.Kafka.Topics...).
	//	WithGroupID(cfg.Kafka.GroupID)
	//
	//kafkaConsumer := consumer.New(kafkaCfg, log)
	//defer kafkaConsumer.Close()
	//
	//kafkaHandler := kafkahandler.NewHandler(container.GamificationService, log)
	//
	//// Context with cancel for graceful shutdown
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	//
	//// Handle shutdown signals
	//go func() {
	//	sigCh := make(chan os.Signal, 1)
	//	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	//	<-sigCh
	//	log.Info("Shutdown signal received")
	//	cancel()
	//}()
	//
	//// Run all servers with errgroup
	//g, gCtx := errgroup.WithContext(ctx)
	//
	//
	//// Kafka consumer
	//g.Go(func() error {
	//	log.Info("Starting Kafka consumer",
	//		zap.Strings("brokers", cfg.Kafka.Brokers),
	//		zap.Strings("topics", cfg.Kafka.Topics),
	//		zap.String("group_id", cfg.Kafka.GroupID),
	//	)
	//	return kafkaConsumer.Run(gCtx, kafkaHandler.HandleMessage)
	//})
	//
	//if err := g.Wait(); err != nil {
	//	log.Error("Server error", zap.Error(err))
	//}

	if err := runner.Run(log, httpServer, grpcServer); err != nil {
		log.Fatal("Failed to start gamification service", zap.Error(err))
	}

	log.Info("Gamification Service stopped")
}
