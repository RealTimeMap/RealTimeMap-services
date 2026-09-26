package main

import (
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	markstat "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	grpcserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/grpc"
	httpserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/userdeleted"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/like"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/infrastructure/persistence/postgres"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "mark_action-service")
	defer log.Sync()

	log.Info("Starting Mark Service", zap.String("env", cfg.Env))

	db := database.MustNew(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}, log)
	defer database.Close(db)
	err := db.AutoMigrate(&like.Reaction{}, &mark.Mark{}, &category.Category{}, &personal.Group{}, &personal.Model{}, &personal.Revision{})
	if err != nil {
		log.Fatal("Failed to migrate likes", zap.Error(err))
	}
	container := app.MustContainer(cfg, db, log)

	httpServer := httpserver.NewServer(cfg.Http, log)
	httpServer.Router().Static("/store", "./store")
	http.RegisterRoutes(httpServer.Router(), container)

	grpcServer, err := grpcserver.NewServer(cfg.GrpcServer, log, func(s *grpc.Server) {
		markstat.RegisterMarkStatsServiceServer(s, container.MarkStatServer)
	})
	if err != nil {
		log.Fatal("Failed to start Mark Service", zap.Error(err))
	}

	servers := []runner.Server{httpServer, grpcServer}
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
		// Без консьюмера удаление аккаунта не сотрёт метки — это нарушение,
		// а не штатный режим, поэтому громко.
		log.Warn("Kafka consumer disabled: user.deleted will not be processed")
	}

	if err := runner.Run(log, servers...); err != nil {
		log.Error("Server error", zap.Error(err))
	}

	log.Info("Mark Service stopped")
}
