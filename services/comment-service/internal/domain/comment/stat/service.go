package stat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
	"go.uber.org/zap"
)

type Service struct {
	statRepo Repository

	logger *zap.Logger
}

func NewService(statRepo Repository, logger *zap.Logger) *Service {
	return &Service{
		statRepo: statRepo,
		logger:   logger,
	}
}

func (s *Service) GetStat(ctx context.Context, userID uint, params date.Resolved) (int64, int64, error) {
	c1, c2, err := s.statRepo.GetCountsByPeriod(ctx, userID, params)
	if err != nil {
		s.logger.Error("GetStat err", zap.Error(err))
		return 0, 0, err
	}
	return c1, c2, nil
}
