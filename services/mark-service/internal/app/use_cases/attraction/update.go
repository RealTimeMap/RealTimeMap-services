package attraction

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/attraction"
)

type Updater interface {
	Update(ctx context.Context, params srv.UpdateAttractionParams) (srv.Model, error)
}

// UpdateAttractionCommand — nil означает "не менять"; Slides заменяются целиком
type UpdateAttractionCommand struct {
	ID       uuid.UUID
	Slug     *string
	Name     *string
	City     *string
	Category *string
	Status   *string
	Geom     *types.Point
	Slides   *[]CreateSlideCommand
}

type UpdateAttractionHandler struct {
	updater Updater

	logger *zap.Logger
}

func NewUpdateAttractionHandler(updater Updater, logger *zap.Logger) *UpdateAttractionHandler {
	return &UpdateAttractionHandler{
		updater: updater,
		logger:  logger,
	}
}

func (h *UpdateAttractionHandler) Handle(ctx context.Context, cmd UpdateAttractionCommand) (AttractionResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.UpdateAttraction"))

	params := srv.UpdateAttractionParams{
		ID:       cmd.ID,
		Slug:     cmd.Slug,
		Name:     cmd.Name,
		City:     cmd.City,
		Category: cmd.Category,
		Status:   cmd.Status,
		Geom:     cmd.Geom,
	}
	if cmd.Slides != nil {
		slides := toSlideParams(*cmd.Slides)
		params.Slides = &slides
	}

	obj, err := h.updater.Update(ctx, params)
	if err != nil {
		return AttractionResult{}, err
	}
	return toAttractionResult(obj), nil
}
