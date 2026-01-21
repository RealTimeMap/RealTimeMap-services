package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgUserExpHistoryRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgUserExpHistoryRepository(db *gorm.DB, logger *zap.Logger) repository.UserExpHistoryRepository {
	return &PgUserExpHistoryRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgUserExpHistoryRepository) Create(ctx context.Context, userExpHistory *model.UserExpHistory) (*model.UserExpHistory, error) {
	r.logger.Info("start PgUserExpHistoryRepository.Create")
	err := r.db.WithContext(ctx).Model(model.UserExpHistory{}).Create(&userExpHistory).Error
	if err != nil {
		return nil, err
	}
	return userExpHistory, nil
}

func (r *PgUserExpHistoryRepository) CountForDay(ctx context.Context, userID, configID uint, date time.Time) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.UserExpHistory{}).Where("user_id = ? AND DATE(created_at) = DATE(?) AND config_id = ?", userID, date, configID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PgUserExpHistoryRepository) ExistsBySourceID(ctx context.Context, userID, configID, sourceID uint) (bool, error) {
	var exist bool
	err := r.db.WithContext(ctx).Model(&model.UserExpHistory{}).Select("1").
		Where("user_id = ?", userID).
		Where("config_id = ?", configID).
		Where("source_id = ?", sourceID).
		Limit(1).First(&exist).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return exist, nil
}
