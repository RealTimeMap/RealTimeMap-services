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

type PgXPRewardRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgXPRewardRepository(db *gorm.DB, logger *zap.Logger) repository.XPRewardRepository {
	return &PgXPRewardRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgXPRewardRepository) GetByID(ctx context.Context, id uint) (*model.XPReward, error) {
	var achievement *model.XPReward

	err := r.db.WithContext(ctx).Model(&achievement).Where("id = ?", id).First(&achievement).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Warn("Не айдено")
			return nil, domainerrors.XPRewardNotFoundError(id)
		}
		return nil, err
	}
	return achievement, nil
}
