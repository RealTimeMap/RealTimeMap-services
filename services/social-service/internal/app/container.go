package app

import (
	pkgprogress "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/progress"
	pkgmark "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/stats/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/blockeduser"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	progressadapter "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/grpc/progress"
	markstatadapter "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/grpc/stats"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/persistence/postgres"
	profilegrpc "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/grpc/profile"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	ProfileRepo        repository.ProfileRepository
	ProfileService     *profile.Service
	ProfileStatService *profile.StatService
	ProfileGRPCHandler *profilegrpc.Handler

	BlockedUserRepo    repository.BlockedUserRepository
	BlockedUserService *blockeduser.Service

	Storage storage.Storage

	ProgressClient *pkgprogress.Client
	MarkStatClient *pkgmark.Client

	Logger *zap.Logger
	DB     *gorm.DB
}

func (c *Container) Close() error {
	if c.ProgressClient != nil {
		return c.ProgressClient.Close()
	}
	return nil
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {
	store, err := storage.NewLocalStorage(cfg.Storage.BasePath, cfg.Storage.BaseURL, logger)
	if err != nil {
		panic(err)
	}
	photoValidator := mediavalidator.NewPhotoValidator()

	var (
		progressClient *pkgprogress.Client
		progressPort   profile.ProgressGetter
		markStatClient *pkgmark.Client
		markStatPort   profile.MarkStatGetter
	)
	if cfg.Gamification.Address != "" {
		c, err := pkgprogress.NewClient(&cfg.Gamification)
		if err != nil {
			logger.Warn("gamification client init failed, continuing without progress",
				zap.Error(err))
		} else {
			progressClient = c
			progressPort = progressadapter.NewAdapter(c)
		}
	}
	if cfg.MarkStat.Address != "" {
		c, err := pkgmark.NewClient(cfg.MarkStat)
		if err != nil {
			logger.Warn("mark client init failed, continuing without progress", zap.Error(err))
		} else {
			markStatClient = c
			markStatPort = markstatadapter.NewAdapter(markStatClient)
		}
	}

	profileRepo := postgres.NewPgProfileRepository(db, logger)
	profileService := profile.NewProfileService(profileRepo, store, photoValidator, progressPort, logger)
	profileStatService := profile.NewStatService(markStatPort, logger)
	profileHandler := profilegrpc.NewHandler(profileService, logger)

	blockedUserRepo := postgres.NewPgBlockedUserRepository(db, logger)
	blockedUserService := blockeduser.NewService(blockedUserRepo, profileRepo, logger)

	return &Container{
		ProfileRepo:        profileRepo,
		ProfileService:     profileService,
		ProfileStatService: profileStatService,
		ProfileGRPCHandler: profileHandler,

		BlockedUserRepo:    blockedUserRepo,
		BlockedUserService: blockedUserService,

		Storage: store,

		ProgressClient: progressClient,

		Logger: logger,
		DB:     db,
	}
}
