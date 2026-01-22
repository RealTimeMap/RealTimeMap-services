package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/domainerrors"
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

func (r *PgUserProgressRepository) GetOrCreate(ctx context.Context, userID uint) (*model.UserProgress, error) {
	progress := &model.UserProgress{UserID: userID}

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Preload("Level").FirstOrCreate(progress).Error

	return progress, err
}

func (r *PgUserProgressRepository) Create(ctx context.Context, user *model.UserProgress) (*model.UserProgress, error) {
	err := r.db.WithContext(ctx).Create(&user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrProgressNotFount(userID)
		}
		return nil, err
	}
	return progress, nil
}

func (r *PgUserProgressRepository) GetTopUsers(ctx context.Context) ([]*model.UserProgress, error) {
	var users []*model.UserProgress
	err := r.db.WithContext(ctx).Model(&model.UserProgress{}).Order("current_level DESC").Limit(10).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
