package stats

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"go.uber.org/zap"
)

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
