package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
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

func (r *MarkStatRepository) GetCountForMonths(ctx context.Context, userID uint, year int) ([]model.MonthlyActivity, error) {
	r.log.Info("start MarkStatRepository.GetCountForMonths", zap.Uint("user_id", userID))

	type result struct {
		Month int
		Count int64
	}

	var rows []result

	err := r.db.WithContext(ctx).Model(&model.Mark{}).
		Select("EXTRACT(MONTH FROM created_at)::int AS month, COUNT(*) AS count").
		Where("user_id = ?", userID).
		Where("EXTRACT(YEAR FROM created_at) = ?", year).
		Group("month").
		Order("month ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[int]int64, len(rows))

	for _, row := range rows {
		counts[row.Month] = row.Count
	}

	months := date.GetMonthsName()

	activity := make([]model.MonthlyActivity, 12)
	for i := 0; i < 12; i++ {
		activity[i] = model.MonthlyActivity{
			Month: months[i],
			Count: counts[i+1],
		}
	}
	return activity, nil

}
