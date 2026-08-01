package bug

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"go.uber.org/zap"
)

type Creator interface {
	Create(ctx context.Context, data bug.CreateBugParams) error
}

type CreatorBugHandler struct {
	creator Creator

	logger *zap.Logger
}

func NewCreatorBugHandler(creator Creator, logger *zap.Logger) *CreatorBugHandler {
	return &CreatorBugHandler{creator: creator, logger: logger}
}

type ApplicationInfoCommand struct {
	Build string
	Logs  []string
}

type DeviceInfoCommand struct {
	OS         string
	Platform   string
	Resolution string
	Battery    *float64
}

type CreateBugCommand struct {
	Title, Desc string
	Tag         string
	UserID      *uint
	App         ApplicationInfoCommand
	Device      DeviceInfoCommand
}

func (h *CreatorBugHandler) Handle(ctx context.Context, cmd CreateBugCommand) error {
	err := h.creator.Create(ctx, bug.CreateBugParams{
		Tag: cmd.Tag,
		App: bug.ApplicationInfoParams{
			Build: cmd.App.Build,
			Logs:  cmd.App.Logs,
		},
		Title:  cmd.Title,
		Desc:   cmd.Desc,
		UserID: cmd.UserID,
		Device: bug.DeviceInfoParams{
			OS:         cmd.Device.OS,
			Platform:   cmd.Device.Platform,
			Resolution: cmd.Device.Resolution,
			Battery:    cmd.Device.Battery,
		},
	})
	if err != nil {
		h.logger.Error("create bug", zap.Error(err))
		return err
	}
	return nil
}
