package postgres

import (
	"context"
	"time"

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

func (r *MarkStatRepository) GetCountPerPeriod(ctx context.Context, userID uint, start, end time.Time) ([]model.DayActivity, error) {
	r.log.Info("start MarkStatRepository.GetCountPerPeriod", zap.Uint("user_id", userID))

	loc := start.Location()
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)

	type result struct {
		Day   time.Time
		Count int64
	}
	var rows []result

	err := r.db.WithContext(ctx).Model(&model.Mark{}).
		Select("created_at::date AS day, COUNT(*) AS count").
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startDay, endDay.AddDate(0, 0, 1)).
		Group("day").
		Order("day ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[time.Time]int64, len(rows))
	for _, row := range rows {
		key := time.Date(row.Day.Year(), row.Day.Month(), row.Day.Day(), 0, 0, 0, 0, loc)
		counts[key] = row.Count
	}

	days := int(endDay.Sub(startDay).Hours()/24) + 1
	activity := make([]model.DayActivity, 0, days)
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		activity = append(activity, model.DayActivity{
			Day:   d,
			Count: counts[d],
		})
	}
	return activity, nil
}

func (r *MarkStatRepository) GetPopularCategories(ctx context.Context, userID uint) ([]model.CategoryStat, error) {
	var rows []model.CategoryStat
	err := r.db.WithContext(ctx).
		Table("marks AS m").
		Select("c.id AS category_id, c.category_name, COUNT(*) AS count").
		Joins("JOIN categories c ON c.id = m.category_id").
		Where("m.user_id = ? AND m.deleted_at IS NULL", userID).
		Group("c.id, c.category_name").
		Order("count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
