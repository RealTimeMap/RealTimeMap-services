package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/cache"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/gamificationservice"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelgenerator"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelservice"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/progressservice"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/infrastructure/persistence/postgres"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	GamificationService *gamificationservice.GamificationService
	LevelService        *levelservice.LevelService
	ProgressRepo        repository.UserProgressRepository
	ProgressService     *progressservice.ProgressService

	CacheStrategy cache.Cache
	Logger        *zap.Logger
}

func NewContainer(config *config.Config, db *gorm.DB, logger *zap.Logger) *Container {
	progressRepo := postgres.NewPgUserProgressRepository(db, logger)
	levelRepo := postgres.NewPgLevelRepository(db, logger)
	configRepo := postgres.NewPgEventConfigRepository(db, logger)
	userHistoryRepo := postgres.NewPgUserExpHistoryRepository(db, logger)

	cli := redis.NewClient(&redis.Options{Addr: config.Redis.Host})

	cacheStrategy := getCacheStrategy(config.CacheStrategy, logger, cli)
	strategy := levelgenerator.NewLinearGenerator()

	progressService := progressservice.NewProgressService(progressRepo, logger)
	levelService := levelservice.NewLevelService(levelRepo, strategy, logger)
	gamificationService := gamificationservice.NewGamificationService(levelService, configRepo, progressRepo, userHistoryRepo, logger)

	return &Container{
		GamificationService: gamificationService,
		LevelService:        levelService,
		ProgressRepo:        progressRepo,

		ProgressService: progressService,

		CacheStrategy: cacheStrategy,
		Logger:        logger,
	}
}

func getCacheStrategy(strategy string, logger *zap.Logger, cli *redis.Client) cache.Cache {
	switch strategy {
	case "memory":
		logger.Info("choice memory cache")
		return cache.NewMemoryCache()
	case "redis":
		logger.Info("choice redis cache")
		if cli != nil {
			return cache.NewRedisCache(cli, logger)
		} else {
			return cache.NewMemoryCache()
		}
	default:
		logger.Info("choice memory cache")
		return cache.NewMemoryCache()
	}
}
