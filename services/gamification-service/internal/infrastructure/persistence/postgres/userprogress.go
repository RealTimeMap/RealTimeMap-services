package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgUserProgressRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPgUserProgressRepository(db *gorm.DB, logger *zap.Logger) repository.UserProgressRepository {
	return &PgUserProgressRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgUserProgressRepository) Update(ctx context.Context, user *model.UserProgress) (*model.UserProgress, error) {
	err := r.db.WithContext(ctx).Model(&model.UserProgress{}).Where("user_id = ?", user.UserID).Save(&user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *PgUserProgressRepository) GetByID(ctx context.Context, userID uint) (*model.UserProgress, error) {
	var progress *model.UserProgress
	err := r.db.WithContext(ctx).First(&progress, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return progress, nil
}
