package stats

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/repository"
	"go.uber.org/zap"
)

type CommentStatsService struct {
	statRepo repository.StatisticRepository

	logger *zap.Logger
}

func NewCommentStatsService(statRepo repository.StatisticRepository, logger *zap.Logger) *CommentStatsService {
	return &CommentStatsService{
		statRepo: statRepo,
		logger:   logger,
	}
}

func (s *CommentStatsService) GetStat(ctx context.Context, userID uint, params date.Resolved) (int64, int64, error) {

	c1, c2, err := s.statRepo.GetCountsByPeriod(ctx, userID, params)
	if err != nil {
		s.logger.Error("GetStat err", zap.Error(err))
		return 0, 0, err
	}
	return c1, c2, nil
}
