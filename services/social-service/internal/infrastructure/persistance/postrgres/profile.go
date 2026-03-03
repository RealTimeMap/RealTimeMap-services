package postrgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgProfileRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgProfileRepository(db *gorm.DB, logger *zap.Logger) repository.ProfileRepository {
	return &PgProfileRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgProfileRepository) Create(ctx context.Context, profile *model.Profile) (*model.Profile, error) {
	err := r.db.WithContext(ctx).Create(profile).Error
	if err != nil {
		return nil, err
	}
	return profile, nil
}
