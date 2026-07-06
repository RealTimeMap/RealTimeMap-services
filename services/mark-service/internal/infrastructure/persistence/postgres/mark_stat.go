package postgres

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgMarkStatRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgMarkStatRepository(db *gorm.DB, logger *zap.Logger) mark.StatsRepository {
	return &PgMarkStatRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgMarkStatRepository) GetMarkCount(ctx context.Context, userID uint) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&mark.Mark{}).
		Where("user_id = ?", userID).Count(&count).Error

	return count, err
}

func (r *PgMarkStatRepository) GetCountForMonths(ctx context.Context, userID uint, year int) ([]mark.MonthlyActivity, error) {
	r.logger.Info("start PgMarkStatRepository.GetCountForMonths", zap.Uint("user_id", userID))

	type result struct {
		Month int
		Count int64
	}

	var rows []result

	err := r.db.WithContext(ctx).Model(&mark.Mark{}).
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

	activity := make([]mark.MonthlyActivity, 12)
	for i := 0; i < 12; i++ {
		activity[i] = mark.MonthlyActivity{
			Month: months[i],
			Count: counts[i+1],
		}
	}
	return activity, nil

}

func (r *PgMarkStatRepository) GetCountPerPeriod(ctx context.Context, userID uint, start, end time.Time) ([]mark.DayActivity, error) {
	r.logger.Info("start PgMarkStatRepository.GetCountPerPeriod", zap.Uint("user_id", userID))

	loc := start.Location()
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)

	type result struct {
		Day   time.Time
		Count int64
	}
	var rows []result

	err := r.db.WithContext(ctx).Model(&mark.Mark{}).
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
	activity := make([]mark.DayActivity, 0, days)
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		activity = append(activity, mark.DayActivity{
			Day:   d,
			Count: counts[d],
		})
	}
	return activity, nil
}

func (r *PgMarkStatRepository) GetPopularCategories(ctx context.Context, userID uint) ([]category.CategoryStat, error) {
	var rows []category.CategoryStat
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
