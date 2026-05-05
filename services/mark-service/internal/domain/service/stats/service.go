package stats

import (
	"context"

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
