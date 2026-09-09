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
	DB       *gorm.DB

	// ServiceApiKey — ключ межсервисных маршрутов. Транспорт берёт его
	// отсюда, чтобы не тянуть весь конфиг в слой маршрутов.
	ServiceApiKey string
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) (*Container, error) {
	bugRepo := postgres.NewPgBugRepository(db, logger)
	bugService := bug.NewService(bugRepo, logger)
	bugUseCases := &bugcases.Application{
		Create: bugcases.NewCreatorBugHandler(bugService, logger),
		List:   bugcases.NewListBugHandler(bugService, logger),
		Get:    bugcases.NewGetBugHandler(bugService, logger),
		Link:   bugcases.NewLinkBugHandler(bugService, logger),
	}

	return &Container{
		BugCases:      bugUseCases,
		Logger:        logger,
		DB:            db,
		ServiceApiKey: cfg.ServiceApiKey,
	}, nil
}
