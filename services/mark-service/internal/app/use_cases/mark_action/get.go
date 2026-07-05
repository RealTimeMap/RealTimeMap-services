package mark_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

const mapZoomThreshold = 12

type MarkGetter interface {
	GetMarksInArea(ctx context.Context, filter mark.Filter) ([]*mark.Mark, error)
	GetMarksInCluster(ctx context.Context, filter mark.Filter) ([]*mark.Cluster, error)
	//GetDetailMark(ctx context.Context, markID uint) (*mark.Mark, error)
}

type GetterMarkHandler struct {
	getter MarkGetter

	logger *zap.Logger
}

func NewMarkGetterHandler(getter MarkGetter, logger *zap.Logger) *GetterMarkHandler {
	return &GetterMarkHandler{getter: getter, logger: logger}
}

func (h *GetterMarkHandler) Handle(ctx context.Context, filter mark.Filter) (MapViewResult, error) {
	if filter.ZoomLevel < mapZoomThreshold {
		h.logger.Debug("Getting marks in cluster", zap.Any("filter", filter))
		clusters, err := h.getter.GetMarksInCluster(ctx, filter)
		if err != nil {
			return MapViewResult{}, err
		}
		return MapViewResult{Mode: ModeCluster, Clusters: toClusterResult(clusters)}, nil
	}
	h.logger.Debug("Getting marks in area", zap.Any("filter", filter))
	marks, err := h.getter.GetMarksInArea(ctx, filter)
	if err != nil {
		return MapViewResult{}, err
	}
	return MapViewResult{Mode: ModeMarks, Marks: toMultiMarkResult(marks)}, nil

}
