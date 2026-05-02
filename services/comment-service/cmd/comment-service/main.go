package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	httpserver "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	httptransport "github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/transport/http"
	"go.uber.org/zap"
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
	db.AutoMigrate(&model.Comment{}, &model.Reaction{})

	container := app.NewContainer(cfg, db, log)
	defer container.Close()

	httpServer := httpserver.NewServer(cfg.HTTP, log)
	httptransport.RegisterRoutes(httpServer.Router(), container)

	if err := runner.Run(log, httpServer); err != nil {
		log.Fatal("Comment Service error", zap.Error(err))
	}

	log.Info("Comment Service stopped")
}
