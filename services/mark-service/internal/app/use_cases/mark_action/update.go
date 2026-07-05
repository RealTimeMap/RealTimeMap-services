package mark_action

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type Updater interface {
	UpdateMark(ctx context.Context, input mark.UpdateMarkParams, userID, markID uint) (*mark.Mark, error)
}

type MarkUpdateCommand struct {
	UserID uint
	MarkID uint
	// Поля метки
	MarkName       string
	AdditionalInfo *string
	EndAt          *time.Time
	PhotosToDelete []string
	Photos         []mediavalidator.PhotoInput
}

type UpdateMarkHandler struct {
	updater Updater

	logger *zap.Logger
}

func NewUpdateMarkHandler(updater Updater, logger *zap.Logger) *UpdateMarkHandler {
	return &UpdateMarkHandler{
		updater: updater,
		logger:  logger,
	}
}

func (h *UpdateMarkHandler) Handle(ctx context.Context, cmd MarkUpdateCommand) (MarkResult, error) {
	h.logger.Info("start markUseCases.UpdateMarkHandler.Handle",
		zap.Uint("markID", cmd.MarkID), zap.Uint("userID", cmd.UserID))

	res, err := h.updater.UpdateMark(ctx, mark.UpdateMarkParams{
		MarkName:       cmd.MarkName,
		AdditionalInfo: cmd.AdditionalInfo,
		EndAt:          cmd.EndAt,
		PhotosToDelete: cmd.PhotosToDelete,
		Photos:         cmd.Photos,
	}, cmd.UserID, cmd.MarkID)
	if err != nil {
		h.logger.Error("failed to update mark", zap.Error(err))
		return MarkResult{}, err
	}

	return toMarkResult(res), nil
}
