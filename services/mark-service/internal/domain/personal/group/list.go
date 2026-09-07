package group

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"go.uber.org/zap"
)

func (s *Service) List(ctx context.Context, userID uint, params pagination.Params) ([]*Model, int64, error) {
	s.logger.Info("start List", zap.String("layer", "domain service"))

	return s.repo.List(ctx, userID, params)
}
