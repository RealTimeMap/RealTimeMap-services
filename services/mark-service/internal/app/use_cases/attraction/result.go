package attraction

import (
	"time"

	"github.com/google/uuid"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/attraction"
)

type SlideBlockResult struct {
	ID        uuid.UUID
	Position  uint
	BlockType string
	Content   map[string]any
}

type SlideResult struct {
	ID       uuid.UUID
	Position uint
	Duration uint
	Blocks   []SlideBlockResult
}

type AttractionResult struct {
	ID        uuid.UUID
	Slug      string
	Name      string
	City      string
	Category  string
	Status    string
	Geom      types.Point
	Slides    []SlideResult
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AttractionShortResult — элемент списка: только то, что нужно для точки на карте
type AttractionShortResult struct {
	ID       uuid.UUID
	Slug     string
	Name     string
	Category string
	Geom     types.Point
}

func toAttractionShortResult(obj srv.Model) AttractionShortResult {
	return AttractionShortResult{
		ID:       obj.ID,
		Slug:     obj.Slug,
		Name:     obj.Name,
		Category: obj.Category,
		Geom:     obj.Geom,
	}
}

func toAttractionResult(obj srv.Model) AttractionResult {
	slides := make([]SlideResult, 0, len(obj.Slides))
	for _, s := range obj.Slides {
		blocks := make([]SlideBlockResult, 0, len(s.Blocks))
		for _, b := range s.Blocks {
			blocks = append(blocks, SlideBlockResult{
				ID:        b.ID,
				Position:  b.Position,
				BlockType: string(b.BlockType),
				Content:   b.Content,
			})
		}
		slides = append(slides, SlideResult{
			ID:       s.ID,
			Position: s.Position,
			Duration: s.DurationMs,
			Blocks:   blocks,
		})
	}

	return AttractionResult{
		ID:        obj.ID,
		Slug:      obj.Slug,
		Name:      obj.Name,
		City:      obj.City,
		Category:  obj.Category,
		Status:    string(obj.Status),
		Geom:      obj.Geom,
		Slides:    slides,
		CreatedAt: obj.CreatedAt,
		UpdatedAt: obj.UpdatedAt,
	}
}
