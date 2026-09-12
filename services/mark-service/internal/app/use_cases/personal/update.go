package personal

import (
	"context"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type Updater interface {
	Update(ctx context.Context, params srv.UpdatePersonalMarkParams, markID, userID uint) (*srv.Model, error)
}

type UpdatePersonalHandler struct {
	updater Updater

	logger *zap.Logger
}

func NewUpdatePersonalHandler(updater Updater, logger *zap.Logger) *UpdatePersonalHandler {
	return &UpdatePersonalHandler{
		updater: updater,
		logger:  logger,
	}
}

type UpdatePersonalMarkCommand struct {
	MarkID uint
	UserID uint

	Geom *types.Point

	Title       *string
	Description *string
	Category    *string
	Color       *string
	Icon        *string

	IsVisible *bool

	GroupsIds []uint

	PhotosToDelete []string
	Photos         []mediavalidator.PhotoInput
}

func (h *UpdatePersonalHandler) Handle(ctx context.Context, cmd UpdatePersonalMarkCommand) (PersonalMarkResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Update"),
		zap.Uint("markID", cmd.MarkID), zap.Uint("userID", cmd.UserID))

	obj, err := h.updater.Update(ctx, srv.UpdatePersonalMarkParams{
		Geom:           cmd.Geom,
		Title:          cmd.Title,
		Description:    cmd.Description,
		Category:       cmd.Category,
		Color:          cmd.Color,
		Icon:           cmd.Icon,
		IsVisible:      cmd.IsVisible,
		GroupsIds:      cmd.GroupsIds,
		PhotosToDelete: cmd.PhotosToDelete,
		Photos:         cmd.Photos,
	}, cmd.MarkID, cmd.UserID)
	if err != nil {
		return PersonalMarkResult{}, err
	}

	return toPersonalMarkResult(obj), nil
}
