package group

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	groupsrv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal/group"
	"go.uber.org/zap"
)

type ListGetter interface {
	List(ctx context.Context, userID uint, params pagination.Params) ([]*groupsrv.Model, int64, error)
}

type ListGroupHandler struct {
	getter ListGetter

	logger *zap.Logger
}

type GetListCommand struct {
	Params pagination.Params
	UserID uint
}

func NewListGroupHandler(getter ListGetter, logger *zap.Logger) *ListGroupHandler {
	return &ListGroupHandler{
		getter: getter,
		logger: logger,
	}
}

func (h *ListGroupHandler) Handle(ctx context.Context, cmd GetListCommand) ([]GroupResult, int64, error) {
	h.logger.Info("start Handle List", zap.String("layer", "use_case"))

	objs, count, err := h.getter.List(ctx, cmd.UserID, cmd.Params)
	if err != nil {
		return []GroupResult{}, 0, err
	}

	response := toListGroupResult(objs)
	return response, count, nil
}
