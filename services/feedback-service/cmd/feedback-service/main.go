package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/runner"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
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

	httpServer := http.NewServer(cfg.Http, log)
	httpTransport.RegisterRoutes(httpServer.Router(), container)

	if err := runner.Run(log, httpServer); err != nil {
		log.Fatal("Failed to start feedback service", zap.Error(err))
	}

}
