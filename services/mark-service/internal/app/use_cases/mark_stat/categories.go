package mark_stat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"go.uber.org/zap"
)

const topN int = 5

type CategoryStatGetter interface {
	GetPopularUserCategories(ctx context.Context, userID uint, topN int) ([]category.CategoryStat, error)
}

type MarkStatCategoryHandler struct {
	getter CategoryStatGetter

	logger *zap.Logger
}

func NewMarkStatCategoryHandler(getter CategoryStatGetter, logger *zap.Logger) *MarkStatCategoryHandler {
	return &MarkStatCategoryHandler{
		getter: getter,
		logger: logger,
	}
}

func (s *MarkStatCategoryHandler) Handle(ctx context.Context, userID uint, topN int) ([]CategoryStatResult, error) {
	objs, err := s.getter.GetPopularUserCategories(ctx, userID, topN)
	if err != nil {
		s.logger.Error("GetPopularUserCategories", zap.Error(err))
		return []CategoryStatResult{}, err
	}
	return toMultiCategoryStatResult(objs), nil
}
