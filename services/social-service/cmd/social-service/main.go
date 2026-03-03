package main

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad()
	log := logger.MustNewByEnv(cfg.Env, "social-service")
	defer log.Sync()

	log.Info("Starting social service", zap.String("env", cfg.Env))

	// Database
	db := database.MustNew(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
	}, log)
	defer database.Close(db)

	db.AutoMigrate(&model.Profile{}, &model.Friendship{})
}
