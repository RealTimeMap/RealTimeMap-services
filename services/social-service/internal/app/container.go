package app

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/persistence/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	ProfileRepo    repository.ProfileRepository
	ProfileService *profile.Service

	Logger *zap.Logger
}

func NewContainer(db *gorm.DB, logger *zap.Logger) *Container {
	profileRepo := postgres.NewPgProfileRepository(db, logger)

	profileService := profile.NewProfileService(profileRepo, logger)

	return &Container{
		ProfileRepo:    profileRepo,
		ProfileService: profileService,

		Logger: logger,
	}
}
