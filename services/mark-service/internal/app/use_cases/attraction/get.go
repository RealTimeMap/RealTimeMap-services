package attraction

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/attraction"
)

type Getter interface {
	Get(ctx context.Context, id uuid.UUID) (srv.Model, error)
}

type GetAttractionQuery struct {
	ID uuid.UUID
}

type GetAttractionHandler struct {
	getter Getter

	logger *zap.Logger
}

func NewGetAttractionHandler(getter Getter, logger *zap.Logger) *GetAttractionHandler {
	return &GetAttractionHandler{
		getter: getter,
		logger: logger,
	}
}

func (h *GetAttractionHandler) Handle(ctx context.Context, q GetAttractionQuery) (AttractionResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.GetAttraction"))

	obj, err := h.getter.Get(ctx, q.ID)
	if err != nil {
		return AttractionResult{}, err
	}
	return toAttractionResult(obj), nil
}
