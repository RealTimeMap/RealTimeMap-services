package personal

import (
	"context"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type Creator interface {
	Create(ctx context.Context, params srv.CreatePersonalMarkParams) (*srv.Model, error)
}

type CreatePersonalHandler struct {
	creator Creator

	logger *zap.Logger
}

type CreatePersonalMarkCommand struct {
	UserID uint
	Geom   types.Point

	Title       string
	Description *string
	Category    string
	Color       string
	Icon        string

	IsVisible bool

	GroupsIds []uint
	Photos    []mediavalidator.PhotoInput
}

func NewCreatePersonalHandler(creator Creator, logger *zap.Logger) *CreatePersonalHandler {
	return &CreatePersonalHandler{
		creator: creator,
		logger:  logger,
	}
}

type PersonalMarkResult struct {
	ID     uint
	UserID uint
}

func toPersonalMarkResult(obj *srv.Model) PersonalMarkResult {
	return PersonalMarkResult{
		ID:     obj.ID,
		UserID: obj.UserID,
	}
}

func (h *CreatePersonalHandler) Handle(ctx context.Context, cmd CreatePersonalMarkCommand) (PersonalMarkResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Create"))

	obj, err := h.creator.Create(ctx, srv.CreatePersonalMarkParams(cmd))
	if err != nil {
		return PersonalMarkResult{}, err
	}

	return toPersonalMarkResult(obj), err
}
