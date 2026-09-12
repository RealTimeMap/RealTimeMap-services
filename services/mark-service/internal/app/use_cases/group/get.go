package group

import (
	"context"

	"go.uber.org/zap"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type Getter interface {
	Get(ctx context.Context, groupID, userID uint) (srv.Group, error)
}

type GetGroupHandler struct {
	getter Getter

	logger *zap.Logger
}

func NewGetGroupHandler(getter Getter, logger *zap.Logger) *GetGroupHandler {
	return &GetGroupHandler{
		getter: getter,
		logger: logger,
	}
}

type GetGroupQuery struct {
	GroupID uint
	UserID  uint
}

func (h *GetGroupHandler) Handle(ctx context.Context, query GetGroupQuery) (GroupResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Get"),
		zap.Uint("groupID", query.GroupID), zap.Uint("userID", query.UserID))

	obj, err := h.getter.Get(ctx, query.GroupID, query.UserID)
	if err != nil {
		return GroupResult{}, err
	}

	return toGroupResult(&obj), nil
}
