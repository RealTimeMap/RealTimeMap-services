package app

import (
	bugcases "github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/app/use_cases/bug"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	BugCases *bugcases.Application
	Logger   *zap.Logger
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) (*Container, error) {
	bugRepo := postgres.NewPgBugRepository(db, logger)
	bugService := bug.NewService(bugRepo, logger)
	bugUseCases := &bugcases.Application{
		Create: bugcases.NewCreatorBugHandler(bugService, logger),
		List:   bugcases.NewListBugHandler(bugService, logger),
	}

	return &Container{
		BugCases: bugUseCases,
		Logger:   logger,
	}, nil
}
