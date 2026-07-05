package mark_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type UserMarkGetter interface {
	GetUserMarks(ctx context.Context, userID uint, params pagination.Params) ([]*mark.Mark, int64, error)
}

type UserMarkGetterCommand struct {
	UserID uint
	Params pagination.Params
}

func (c *UserMarkGetterCommand) Validate() error {
	c.Params.Defaults()
	return nil
}

type UserMarkGetterHandler struct {
	getter UserMarkGetter

	logger *zap.Logger
}

func NewUserMarkGetterHandler(getter UserMarkGetter, logger *zap.Logger) *UserMarkGetterHandler {
	return &UserMarkGetterHandler{
		getter: getter,
		logger: logger,
	}
}

func (h *UserMarkGetterHandler) Handle(ctx context.Context, cmd UserMarkGetterCommand) ([]MarkResult, int64, error) {
	if err := cmd.Validate(); err != nil {
		return nil, 0, err
	}

	res, count, err := h.getter.GetUserMarks(ctx, cmd.UserID, cmd.Params)
	if err != nil {
		return nil, 0, err
	}

	return toMultiMarkResult(res), count, nil
}
