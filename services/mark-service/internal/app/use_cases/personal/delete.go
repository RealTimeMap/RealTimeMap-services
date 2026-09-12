package personal

import (
	"context"

	"go.uber.org/zap"
)

type Remover interface {
	Delete(ctx context.Context, markID, userID uint) error
}

type DeletePersonalHandler struct {
	remover Remover

	logger *zap.Logger
}

func NewDeletePersonalHandler(remover Remover, logger *zap.Logger) *DeletePersonalHandler {
	return &DeletePersonalHandler{
		remover: remover,
		logger:  logger,
	}
}

type DeletePersonalMarkCommand struct {
	MarkID uint
	UserID uint
}

func (h *DeletePersonalHandler) Handle(ctx context.Context, cmd DeletePersonalMarkCommand) error {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Delete"),
		zap.Uint("markID", cmd.MarkID), zap.Uint("userID", cmd.UserID))

	return h.remover.Delete(ctx, cmd.MarkID, cmd.UserID)
}
