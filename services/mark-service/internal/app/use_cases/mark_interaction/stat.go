package mark_interaction

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

// MarkStatGetter отдаёт агрегированную статистику метки.
type MarkStatGetter interface {
	GetStat(ctx context.Context, params mark.AccrualParams) (mark.MarkStat, error)
}

type StatCommand struct {
	MarkID uint
	UserID uint
}

type GetStatHandler struct {
	getter MarkStatGetter

	logger *zap.Logger
}

func NewGetStatHandler(getter MarkStatGetter, logger *zap.Logger) *GetStatHandler {
	return &GetStatHandler{
		getter: getter,
		logger: logger,
	}
}

func (h *GetStatHandler) Handle(ctx context.Context, cmd StatCommand) (MarkStatResult, error) {
	stat, err := h.getter.GetStat(ctx, mark.AccrualParams{
		MarkID: cmd.MarkID,
		UserID: cmd.UserID,
	})
	if err != nil {
		return MarkStatResult{}, err
	}
	return toMarkStatResult(stat), nil
}
