package group

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Remover interface {
	DeleteGroup(ctx context.Context, groupID uuid.UUID, userID uint) error
}

type DeleteGroupHandler struct {
	remover Remover

	logger *zap.Logger
}

func NewDeleteGroupHandler(remover Remover, logger *zap.Logger) *DeleteGroupHandler {
	return &DeleteGroupHandler{
		remover: remover,
		logger:  logger,
	}
}

type DeleteGroupCommand struct {
	GroupID uuid.UUID
	UserID  uint
}

func (h *DeleteGroupHandler) Handle(ctx context.Context, cmd DeleteGroupCommand) error {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Delete"),
		zap.String("groupID", cmd.GroupID.String()), zap.Uint("userID", cmd.UserID))

	return h.remover.DeleteGroup(ctx, cmd.GroupID, cmd.UserID)
}
