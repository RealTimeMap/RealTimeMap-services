package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/config"
	"go.uber.org/zap"
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
}
