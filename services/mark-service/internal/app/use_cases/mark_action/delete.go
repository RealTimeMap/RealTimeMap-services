package mark_action

import (
	"context"

	"go.uber.org/zap"
)

type Remover interface {
	DeleteMark(ctx context.Context, userID, markID uint) error
}

type DeleteMarkCommand struct {
	UserID uint
	MarkID uint
}

type RemoverMarkHandler struct {
	remover Remover

	logger *zap.Logger
}

func NewRemoverMarkHandler(remover Remover, logger *zap.Logger) *RemoverMarkHandler {
	return &RemoverMarkHandler{
		remover: remover,
		logger:  logger,
	}
}

func (h *RemoverMarkHandler) Handle(ctx context.Context, cmd DeleteMarkCommand) error {
	err := h.remover.DeleteMark(ctx, cmd.UserID, cmd.MarkID)
	if err != nil {
		return err
	}
	return nil
}
