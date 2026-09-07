package group

import (
	"context"

	"go.uber.org/zap"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal/group"
)

type Creator interface {
	CreateGroup(ctx context.Context, param srv.CreateGroupParams) (*srv.Model, error)
}

type CreateGroupCommand struct {
	Name        string
	Description *string
	UserID      uint
}

type CreateGroupHandler struct {
	creator Creator

	logger *zap.Logger
}

func NewCreateGroupHandler(creator Creator, logger *zap.Logger) *CreateGroupHandler {
	return &CreateGroupHandler{
		creator: creator,
		logger:  logger,
	}
}

func (h *CreateGroupHandler) Handle(ctx context.Context, cmd CreateGroupCommand) (GroupResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.Create"))
	obj, err := h.creator.CreateGroup(ctx, srv.CreateGroupParams{
		Name:        cmd.Name,
		Description: cmd.Description,
		UserID:      cmd.UserID,
	})
	if err != nil {
		return GroupResult{}, err
	}

	return toGroupResult(obj), nil
}
