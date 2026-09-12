package group

import (
	"context"

	"go.uber.org/zap"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type Updater interface {
	UpdateGroup(ctx context.Context, params srv.UpdateGroupParams, groupID, userID uint) (*srv.Group, error)
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
	GroupID uint
	UserID  uint

	Name        *string
	Description *string
}

func (h *UpdateGroupHandler) Handle(ctx context.Context, cmd UpdateGroupCommand) (GroupResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Update"),
		zap.Uint("groupID", cmd.GroupID), zap.Uint("userID", cmd.UserID))

	obj, err := h.updater.UpdateGroup(ctx, srv.UpdateGroupParams{
		Name:        cmd.Name,
		Description: cmd.Description,
	}, cmd.GroupID, cmd.UserID)
	if err != nil {
		return GroupResult{}, err
	}

	return toGroupResult(obj), nil
}
