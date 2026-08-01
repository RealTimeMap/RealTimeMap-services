package bug

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"go.uber.org/zap"
)

type ListGetter interface {
	GetList(ctx context.Context, filter bug.GetBugParams) ([]bug.Model, error)
}

type ListBugHandler struct {
	getter ListGetter

	logger *zap.Logger
}

func NewListBugHandler(srv ListGetter, logger *zap.Logger) *ListBugHandler {
	return &ListBugHandler{
		getter: srv,
		logger: logger,
	}
}

type ListBugCommand struct {
	Pagination pagination.Params
	Tag        *string
	Status     *string
}

func (h *ListBugHandler) Handle(ctx context.Context, cmd ListBugCommand) ([]BugResult, error) {
	objs, err := h.getter.GetList(ctx, bug.GetBugParams{
		Tag:        cmd.Tag,
		Status:     cmd.Status,
		Pagination: cmd.Pagination,
	})
	if err != nil {
		return nil, err
	}
	return toMultiBugResult(objs), nil
}
