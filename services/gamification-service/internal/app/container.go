package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	redispkg "github.com/RealTimeMap/RealTimeMap-backend/pkg/redis"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware/cache"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/achievement"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/event"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/level"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/level/generator"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/progress"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/xp"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/infrastructure/persistence/postgres"
	grpctransport "github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/transport/grpc"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	LevelService    *level.Service
	ProgressRepo    repository.UserProgressRepository
	ProgressService *progress.ProgressService

	EventGamificationService *event.Service
	AchievementService       achievement.Service

	ProgressGrpcHandler *grpctransport.Handler

	CacheStrategy cache.Cache
	DB            *gorm.DB
	Logger        *zap.Logger
}

func NewContainer(config *config.Config, db *gorm.DB, logger *zap.Logger) *Container {
	progressRepo := postgres.NewPgUserProgressRepository(db, logger)
	levelRepo := postgres.NewPgLevelRepository(db, logger)

	cli := redispkg.NewRedisCli(config.Redis)

	cacheStrategy := getCacheStrategy(config.CacheStrategy, logger, cli)
	strategy := levelgenerator.NewLinearGenerator()

	progressService := progress.NewProgressService(progressRepo, logger)
	levelService := level.NewLevelService(levelRepo, strategy, logger)
	opRepo := postgres.NewPgXPOperation(db, logger)
	txManager := txmanager.NewTxManager(db)
	op := xp.NewXPOperator(opRepo, progressRepo, levelService, txManager, logger)

	eventRepo := postgres.NewPPgEventRuleRepository(db, logger)
	userAchCountRepo := postgres.NewPgUserAchievementCountRepository(db, logger)
	eventService := event.NewEventGamificationService(eventRepo, opRepo, userAchCountRepo, op, logger)

	achievementRepo := postgres.NewPgAchievementRepository(db, logger)
	xpRewardRepo := postgres.NewPgXPRewardRepository(db, logger)
	userAchRepo := postgres.NewPgUserAchievementRepository(db, logger)
	store, err := storage.NewLocalStorage(config.Storage.BasePath, config.Storage.BaseURL, logger)
	if err != nil {
		panic(err)
	}

	s := achievement.New(achievementRepo, xpRewardRepo, userAchRepo, op, store, mediavalidator.NewPhotoValidator(), logger)
	grpcHandler := grpctransport.NewHandler(progressRepo, levelService, logger)

	return &Container{
		LevelService: levelService,
		ProgressRepo: progressRepo,

		ProgressService: progressService,

		EventGamificationService: eventService,
		AchievementService:       s,

		ProgressGrpcHandler: grpcHandler,

		CacheStrategy: cacheStrategy,
		DB:            db,
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
