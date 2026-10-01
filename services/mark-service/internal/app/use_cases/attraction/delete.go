package attraction

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Remover interface {
	Delete(ctx context.Context, id uuid.UUID) error
}

type DeleteAttractionCommand struct {
	ID uuid.UUID
}

type DeleteAttractionHandler struct {
	remover Remover

	logger *zap.Logger
}

func NewDeleteAttractionHandler(remover Remover, logger *zap.Logger) *DeleteAttractionHandler {
	return &DeleteAttractionHandler{
		remover: remover,
		logger:  logger,
	}
}

func (h *DeleteAttractionHandler) Handle(ctx context.Context, cmd DeleteAttractionCommand) error {
	h.logger.Info("start Handle", zap.String("layer", "use_case.DeleteAttraction"))
	return h.remover.Delete(ctx, cmd.ID)
}
