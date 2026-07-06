package mark

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"go.uber.org/zap"
)

const OtherCategoryName string = "Other"

type StatsService struct {
	statsRepo StatsRepository

	logger *zap.Logger
}

func NewStatService(statsRepo StatsRepository, logger *zap.Logger) *StatsService {
	return &StatsService{
		statsRepo: statsRepo,
		logger:    logger,
	}
}

func (s *StatsService) GetUserMarksCount(ctx context.Context, userID uint) (int64, error) {
	return s.statsRepo.GetMarkCount(ctx, userID)
}

func (s *StatsService) GetUserMonthlyActivity(ctx context.Context, userID uint, year int) ([]MonthlyActivity, error) {
	s.logger.Info("start MarkStatService.GetUserMonthlyActivity")

	result, err := s.statsRepo.GetCountForMonths(ctx, userID, year)
	if err != nil {
		s.logger.Error("GetUserMonthlyActivity", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *StatsService) GetCountsForPeriod(ctx context.Context, userID uint, start, end time.Time) ([]DayActivity, error) {
	s.logger.Info("start MarkStatService.GetCountsForPeriod")
	return s.statsRepo.GetCountPerPeriod(ctx, userID, start, end)
}

func (s *StatsService) GetPopularUserCategories(ctx context.Context, userID uint, topN int) ([]category.CategoryStat, error) {
	s.logger.Info("start MarkStatService.GetPopularUserCategories")

	counts, err := s.statsRepo.GetPopularCategories(ctx, userID)
	if err != nil {
		s.logger.Error("GetPopularUserCategories", zap.Error(err))
		return nil, err
	}
	return buildCategoryStats(counts, topN), nil

}

func buildCategoryStats(data []category.CategoryStat, topN int) []category.CategoryStat {
	var total int64

	for _, i := range data {
		total += i.Count
	}
	if total == 0 {
		return []category.CategoryStat{}
	}

	result := make([]category.CategoryStat, 0, topN+1)
	var otherCount int64

	for i, c := range data {
		if i < topN {
			result = append(result, category.CategoryStat{
				Count:        c.Count,
				CategoryName: c.CategoryName,
				Percent:      utils.Percent(c.Count, total),
			})
		} else {
			otherCount += c.Count
		}
	}
	if otherCount > 0 {
		result = append(result, category.CategoryStat{
			CategoryName: OtherCategoryName,
			Count:        otherCount,
			Percent:      utils.Percent(otherCount, total),
		})
	}

	return result
}
