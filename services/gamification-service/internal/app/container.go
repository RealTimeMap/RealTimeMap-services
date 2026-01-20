package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/gamificationservice"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelgenerator"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelservice"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	GamificationService *gamificationservice.GamificationService

	Logger *zap.Logger
}

func NewContainer(db *gorm.DB, logger *zap.Logger) *Container {

	progressRepo := postgres.NewPgUserProgressRepository(db, logger)
	levelRepo := postgres.NewPgLevelRepository(db, logger)
	configRepo := postgres.NewPgEventConfigRepository(db, logger)

	strategy := levelgenerator.NewLinearGenerator()
	levelService := levelservice.NewLevelService(levelRepo, strategy, logger)
	gamificationService := gamificationservice.NewGamificationService(levelService, configRepo, progressRepo, logger)
	return &Container{
		GamificationService: gamificationService,

		Logger: logger,
	}
}
