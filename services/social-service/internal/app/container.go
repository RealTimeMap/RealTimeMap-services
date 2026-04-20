package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/blockeduser"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	ProfileRepo    repository.ProfileRepository
	ProfileService *profile.Service

	BlockedUserRepo    repository.BlockedUserRepository
	BlockedUserService *blockeduser.Service

	Logger *zap.Logger
	DB     *gorm.DB
}

func NewContainer(db *gorm.DB, logger *zap.Logger) *Container {
	profileRepo := postgres.NewPgProfileRepository(db, logger)
	profileService := profile.NewProfileService(profileRepo, logger)

	blockedUserRepo := postgres.NewPgBlockedUserRepository(db, logger)
	blockedUserService := blockeduser.NewService(blockedUserRepo, profileRepo, logger)

	return &Container{
		ProfileRepo:    profileRepo,
		ProfileService: profileService,

		BlockedUserRepo:    blockedUserRepo,
		BlockedUserService: blockedUserService,

		Logger: logger,
		DB:     db,
	}
}
