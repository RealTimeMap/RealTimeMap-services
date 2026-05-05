package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	markstat "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	grpcserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/grpc"
	httpserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "mark-service")
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
	db.AutoMigrate(&model.Mark{}, &model.Category{})

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

	if err := runner.Run(log, httpServer, grpcServer); err != nil {
		log.Error("Server error", zap.Error(err))
	}

	log.Info("Mark Service stopped")
}
