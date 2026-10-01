package attraction

import (
	"context"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/attraction"
)

type Creator interface {
	Create(ctx context.Context, params srv.CreateAttractionParams) (srv.Model, error)
}

type CreateSlideBlockCommand struct {
	Position  uint
	BlockType string
	Content   map[string]any
}

type CreateSlideCommand struct {
	Position uint
	Duration uint // В миллисекундах
	Blocks   []CreateSlideBlockCommand
}

type CreateAttractionCommand struct {
	Slug     string
	Name     string
	City     string
	Category string
	Publish  bool
	Geom     types.Point
	Slides   []CreateSlideCommand
}

type CreateAttractionHandler struct {
	creator Creator

	logger *zap.Logger
}

func NewCreateAttractionHandler(creator Creator, logger *zap.Logger) *CreateAttractionHandler {
	return &CreateAttractionHandler{
		creator: creator,
		logger:  logger,
	}
}

func (h *CreateAttractionHandler) Handle(ctx context.Context, cmd CreateAttractionCommand) (AttractionResult, error) {
	h.logger.Info("start Handle", zap.String("layer", "use_case.CreateAttraction"))

	obj, err := h.creator.Create(ctx, srv.CreateAttractionParams{
		Slug:     cmd.Slug,
		Name:     cmd.Name,
		City:     cmd.City,
		Category: cmd.Category,
		Publish:  cmd.Publish,
		Geom:     cmd.Geom,
		Slides:   toSlideParams(cmd.Slides),
	})
	if err != nil {
		return AttractionResult{}, err
	}

	return toAttractionResult(obj), nil
}

func toSlideParams(cmds []CreateSlideCommand) []srv.CreateSlideParams {
	slides := make([]srv.CreateSlideParams, 0, len(cmds))
	for _, s := range cmds {
		blocks := make([]srv.CreateSlideBlockParams, 0, len(s.Blocks))
		for _, b := range s.Blocks {
			blocks = append(blocks, srv.CreateSlideBlockParams{
				Position:  b.Position,
				BlockType: b.BlockType,
				Content:   b.Content,
			})
		}
		slides = append(slides, srv.CreateSlideParams{
			Position: s.Position,
			Duration: s.Duration,
			Blocks:   blocks,
		})
	}
	return slides
}
