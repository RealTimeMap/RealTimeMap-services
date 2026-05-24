package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgUserAchievementRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgUserAchievementRepository(db *gorm.DB, logger *zap.Logger) repository.UserAchievementRepository {
	return &PgUserAchievementRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgUserAchievementRepository) Create(ctx context.Context, userID, achID uint) error {
	ua := &model.UserAchievement{
		UserID:        userID,
		AchievementID: achID,
		UnlockedAt:    time.Now(),
	}

	err := r.db.WithContext(ctx).Create(ua).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domainerrors.AchievementAlreadyUnlockedError(achID)
		}
		return err
	}
	return nil
}

func (r *PgUserAchievementRepository) GetUserAchievements(ctx context.Context, userID uint, params pagination.Params) ([]*model.UserAchievement, int64, error) {
	r.logger.Info("GetUserAchievements", zap.Uint("userId", userID))
	var results []*model.UserAchievement
	var total int64

	subQuery := r.db.
		Table("user_achievements ua2").
		Select("1").
		Joins("JOIN achievements a2 ON a2.id = ua2.achievement_id").
		Where("ua2.user_id = ?", userID).
		Where("a2.id = achievements.next_id")

	baseQuery := r.db.WithContext(ctx).
		Model(&model.UserAchievement{}).
		Joins("JOIN achievements ON achievements.id = user_achievements.achievement_id").
		Where("user_achievements.user_id = ?", userID).
		Where("NOT EXISTS (?)", subQuery)

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := baseQuery.
		Preload("Achievement").
		Preload("Achievement.Reward").
		Preload("Achievement.Next").
		Offset(params.Offset()).
		Limit(params.Limit()).
		Find(&results).Error

	return results, total, err
}
