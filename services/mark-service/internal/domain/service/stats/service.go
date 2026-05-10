package stats

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"go.uber.org/zap"
)

const OtherCategoryName string = "Other"

type MarkStatsService struct {
	statsRepo repository.MarkStatsRepository

	logger *zap.Logger
}

func NewMarkStatsService(statsRepo repository.MarkStatsRepository, logger *zap.Logger) *MarkStatsService {
	return &MarkStatsService{
		statsRepo: statsRepo,
		logger:    logger,
	}
}

func (s *MarkStatsService) GetMarkCount(ctx context.Context, userID uint) (int64, error) {
	return s.statsRepo.GetMarkCount(ctx, userID)
}

func (s *MarkStatsService) GetUserMonthlyActivity(ctx context.Context, userID uint, year int) ([]model.MonthlyActivity, error) {
	s.logger.Info("start MarkStatService.GetUserMonthlyActivity")

	result, err := s.statsRepo.GetCountForMonths(ctx, userID, year)
	if err != nil {
		s.logger.Error("GetUserMonthlyActivity", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *MarkStatsService) GetCountsForPeriod(ctx context.Context, userID uint, start, end time.Time) ([]model.DayActivity, error) {
	s.logger.Info("start MarkStatService.GetCountsForPeriod")
	return s.statsRepo.GetCountPerPeriod(ctx, userID, start, end)
}

func (s *MarkStatsService) GetPopularUserCategories(ctx context.Context, userID uint, topN int) ([]model.CategoryStat, error) {
	s.logger.Info("start MarkStatService.GetPopularUserCategories")

	counts, err := s.statsRepo.GetPopularCategories(ctx, userID)
	if err != nil {
		s.logger.Error("GetPopularUserCategories", zap.Error(err))
		return nil, err
	}
	return buildCategoryStats(counts, topN), nil

}

func buildCategoryStats(data []model.CategoryStat, topN int) []model.CategoryStat {
	var total int64

	for _, i := range data {
		total += i.Count
	}
	if total == 0 {
		return []model.CategoryStat{}
	}

	result := make([]model.CategoryStat, 0, topN+1)
	var otherCount int64

	for i, c := range data {
		if i < topN {
			result = append(result, model.CategoryStat{
				Count:        c.Count,
				CategoryName: c.CategoryName,
				Percent:      utils.Percent(c.Count, total),
			})
		} else {
			otherCount += c.Count
		}
	}
	if otherCount > 0 {
		result = append(result, model.CategoryStat{
			CategoryName: OtherCategoryName,
			Count:        otherCount,
			Percent:      utils.Percent(otherCount, total),
		})
	}

	return result
}
