package group

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type Updater interface {
	UpdateGroup(ctx context.Context, params srv.UpdateGroupParams, groupID uuid.UUID, userID uint) (*srv.Group, error)
}

type UpdateGroupHandler struct {
	updater Updater

	logger *zap.Logger
}

func NewUpdateGroupHandler(updater Updater, logger *zap.Logger) *UpdateGroupHandler {
	return &UpdateGroupHandler{
		updater: updater,
		logger:  logger,
	}
}

type UpdateGroupCommand struct {
	GroupID uuid.UUID
	UserID  uint

	Name        *string
	Description *string
	Color       *string
	Icon        *string
}

func (h *UpdateGroupHandler) Handle(ctx context.Context, cmd UpdateGroupCommand) (GroupResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Update"),
		zap.String("groupID", cmd.GroupID.String()), zap.Uint("userID", cmd.UserID))

	obj, err := h.updater.UpdateGroup(ctx, srv.UpdateGroupParams{
		Name:        cmd.Name,
		Description: cmd.Description,
		Color:       cmd.Color,
		Icon:        cmd.Icon,
	}, cmd.GroupID, cmd.UserID)
	if err != nil {
		return GroupResult{}, err
	}

	return toGroupResult(obj), nil
}
