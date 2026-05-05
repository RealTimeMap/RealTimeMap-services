package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MarkStatRepository struct {
	db    *gorm.DB
	log   *zap.Logger
	layer string
}

func NewMarkStatRepository(db *gorm.DB, logger *zap.Logger) repository.MarkStatsRepository {
	return &MarkStatRepository{
		db:    db,
		log:   logger,
		layer: "MarkStatRepository",
	}
}

func (r *MarkStatRepository) GetMarkCount(ctx context.Context, userID uint) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.Mark{}).
		Where("user_id = ?", userID).Count(&count).Error

	return count, err
}
