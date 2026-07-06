package mark_stat

import (
	"context"

	"go.uber.org/zap"
)

type MarkCountGetter interface {
	GetUserMarksCount(ctx context.Context, userID uint) (int64, error)
}

type MarkCountHandler struct {
	getter MarkCountGetter

	logger *zap.Logger
}

func NewMarkCountHandler(getter MarkCountGetter, logger *zap.Logger) *MarkCountHandler {
	return &MarkCountHandler{
		getter: getter,
		logger: logger,
	}
}

func (h *MarkCountHandler) Handle(ctx context.Context, userID uint) (int64, error) {
	h.logger.Info("Handle mark count", zap.Uint("user_id", userID))
	return h.getter.GetUserMarksCount(ctx, userID)
}
