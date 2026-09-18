package group

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type Creator interface {
	CreateGroup(ctx context.Context, param srv.CreateGroupParams) (*srv.Group, error)
}

type CreateGroupCommand struct {
	// ID — идентификатор от клиента для группы, созданной офлайн.
	// uuid.Nil означает, что его сгенерирует сервер.
	ID          uuid.UUID
	Name        string
	Description *string
	UserID      uint
	Color       string
	Icon        string
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
		ID:          cmd.ID,
		Name:        cmd.Name,
		Description: cmd.Description,
		UserID:      cmd.UserID,
		Color:       cmd.Color,
		Icon:        cmd.Icon,
	})
	if err != nil {
		return GroupResult{}, err
	}

	return toGroupResult(obj), nil
}
