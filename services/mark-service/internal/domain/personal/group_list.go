package personal

import (
	"context"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

func (s *GroupService) List(ctx context.Context, userID uint, params pagination.Params) ([]*Group, int64, error) {
	s.logger.Info("start List", zap.String("layer", "domain service"))

	return s.repo.List(ctx, userID, params)
}

func (s *GroupService) ListChanges(ctx context.Context, userID uint, since *uint, upTo uint, limit int) (Changes[Group], error) {
	objs, err := s.repo.Get(ctx, userID, since, upTo, limit)
	if err != nil {
		return Changes[Group]{}, err
	}

	return toChanges(objs, upTo, limit), nil
}

func (s *GroupService) Name() string {
	return "groups"
}
