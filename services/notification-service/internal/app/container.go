package app

import (
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/infrastructure/persistance/postgres"
)

type Container struct {
	TokenService *token.Service

	DB     *gorm.DB
	Logger *zap.Logger
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {
	tokenRepo := postgres.NewPgUserTokenRepository(db, logger)

	return &Container{
		TokenService: token.NewService(tokenRepo, logger),

		DB:     db,
		Logger: logger,
	}
}
