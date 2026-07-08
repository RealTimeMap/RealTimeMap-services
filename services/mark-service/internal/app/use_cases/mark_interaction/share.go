package mark_interaction

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type ShareCreator interface {
	IncreaseShare(ctx context.Context, params mark.AccrualParams) (int64, error)
}

type ShareCreatorHandler struct {
	creator ShareCreator

	logger *zap.Logger
}

func NewShareCreatorHandler(creator ShareCreator, logger *zap.Logger) *ShareCreatorHandler {
	return &ShareCreatorHandler{
		creator: creator,
		logger:  logger,
	}
}

type ShareCreateCommand struct {
	MarkID uint
	UserID uint
}

func (h *ShareCreatorHandler) Handle(ctx context.Context, cmd ShareCreateCommand) (ShareResult, error) {
	obj, err := h.creator.IncreaseShare(ctx, mark.AccrualParams{
		MarkID: cmd.MarkID,
		UserID: cmd.UserID,
	})
	if err != nil {
		return ShareResult{}, err

	}
	return toShareResult(obj), nil
}
