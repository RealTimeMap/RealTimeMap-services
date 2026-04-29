package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/blockeduser"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/persistence/postgres"
	profilegrpc "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/grpc/profile"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	ProfileRepo        repository.ProfileRepository
	ProfileService     *profile.Service
	ProfileGRPCHandler *profilegrpc.Handler

	BlockedUserRepo    repository.BlockedUserRepository
	BlockedUserService *blockeduser.Service

	Storage storage.Storage

	Logger *zap.Logger
	DB     *gorm.DB
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {
	store, err := storage.NewLocalStorage(cfg.Storage.BasePath, cfg.Storage.BaseURL, logger)
	if err != nil {
		panic(err)
	}
	photoValidator := mediavalidator.NewPhotoValidator()

	profileRepo := postgres.NewPgProfileRepository(db, logger)
	profileService := profile.NewProfileService(profileRepo, store, photoValidator, logger)
	profileHandler := profilegrpc.NewHandler(profileService, logger)

	blockedUserRepo := postgres.NewPgBlockedUserRepository(db, logger)
	blockedUserService := blockeduser.NewService(blockedUserRepo, profileRepo, logger)

	return &Container{
		ProfileRepo:        profileRepo,
		ProfileService:     profileService,
		ProfileGRPCHandler: profileHandler,

		BlockedUserRepo:    blockedUserRepo,
		BlockedUserService: blockedUserService,

		Storage: store,

		Logger: logger,
		DB:     db,
	}
}
