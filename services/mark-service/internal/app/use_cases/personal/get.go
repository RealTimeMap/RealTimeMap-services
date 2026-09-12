package personal

import (
	"context"

	"go.uber.org/zap"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type Getter interface {
	Get(ctx context.Context, markID, userID uint) (srv.Model, error)
}

type GetPersonalHandler struct {
	getter Getter

	logger *zap.Logger
}

func NewGetPersonalHandler(getter Getter, logger *zap.Logger) *GetPersonalHandler {
	return &GetPersonalHandler{
		getter: getter,
		logger: logger,
	}
}

type GetPersonalMarkQuery struct {
	MarkID uint
	UserID uint
}

func (h *GetPersonalHandler) Handle(ctx context.Context, query GetPersonalMarkQuery) (PersonalMarkDetailResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Get"),
		zap.Uint("markID", query.MarkID), zap.Uint("userID", query.UserID))

	obj, err := h.getter.Get(ctx, query.MarkID, query.UserID)
	if err != nil {
		return PersonalMarkDetailResult{}, err
	}

	return toPersonalMarkDetailResult(obj), nil
}
