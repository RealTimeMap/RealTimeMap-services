package mark_stat

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type HeatMapGetter interface {
	GetCountsForPeriod(ctx context.Context, userID uint, start, end time.Time) ([]mark.DayActivity, error)
}

type MarkHeatMapHandler struct {
	getter HeatMapGetter

	logger *zap.Logger
}

func NewMarkHeatMapHandler(getter HeatMapGetter, logger *zap.Logger) *MarkHeatMapHandler {
	return &MarkHeatMapHandler{
		getter: getter,
		logger: logger,
	}
}

func (h *MarkHeatMapHandler) Handle(ctx context.Context, userID uint, start, end time.Time) ([]DayActivityResult, error) {
	objs, err := h.getter.GetCountsForPeriod(ctx, userID, start, end)
	if err != nil {
		h.logger.Error("GetCountsForPeriod", zap.Error(err))
		return nil, err
	}
	return toMultiDayActivityResult(objs), nil
}
