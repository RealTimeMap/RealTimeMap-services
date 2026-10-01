package attraction

import (
	"context"

	"go.uber.org/zap"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/attraction"
)

type CityLister interface {
	ListByCity(ctx context.Context, city string) ([]srv.Model, error)
}

type ListByCityQuery struct {
	City string
}

type ListByCityHandler struct {
	lister CityLister

	logger *zap.Logger
}

func NewListByCityHandler(lister CityLister, logger *zap.Logger) *ListByCityHandler {
	return &ListByCityHandler{
		lister: lister,
		logger: logger,
	}
}

func (h *ListByCityHandler) Handle(ctx context.Context, q ListByCityQuery) ([]AttractionShortResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.ListByCity"))

	objs, err := h.lister.ListByCity(ctx, q.City)
	if err != nil {
		return nil, err
	}

	res := make([]AttractionShortResult, 0, len(objs))
	for _, obj := range objs {
		res = append(res, toAttractionShortResult(obj))
	}
	return res, nil
}
