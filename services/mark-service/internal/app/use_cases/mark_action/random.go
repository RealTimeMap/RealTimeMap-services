package mark_action

import (
	"context"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
)

type RandomGetter interface {
	GetRandomMark(ctx context.Context) (mark.Mark, error)
}

type RandomMarkHandler struct {
	getter RandomGetter

	logger *zap.Logger
}

func NewRandomMarkHandler(getter RandomGetter, logger *zap.Logger) *RandomMarkHandler {
	return &RandomMarkHandler{
		getter: getter,
		logger: logger,
	}
}

func (h *RandomMarkHandler) Handle(ctx context.Context) (MarkResult, error) {
	obj, err := h.getter.GetRandomMark(ctx)
	if err != nil {
		return MarkResult{}, err
	}

	return toMarkResult(&obj), nil
}
