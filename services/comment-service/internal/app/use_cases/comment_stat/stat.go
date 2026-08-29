package comment_stat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
	"go.uber.org/zap"
)

type StatGetter interface {
	GetStat(ctx context.Context, userID uint, params date.Resolved) (int64, int64, error)
}

type GetStatCommand struct {
	UserID uint
	Period date.Resolved
}

type GetStatHandler struct {
	getter StatGetter

	logger *zap.Logger
}

func NewGetStatHandler(getter StatGetter, logger *zap.Logger) *GetStatHandler {
	return &GetStatHandler{getter: getter, logger: logger}
}

func (h *GetStatHandler) Handle(ctx context.Context, cmd GetStatCommand) (StatResult, error) {
	h.logger.Info("start commentStat.GetStatHandler.Handle")

	current, previous, err := h.getter.GetStat(ctx, cmd.UserID, cmd.Period)
	if err != nil {
		return StatResult{}, err
	}
	return toStatResult(current, previous), nil
}
