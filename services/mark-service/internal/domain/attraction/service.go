package attraction

import (
	"context"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

type Service struct {
	repo   Repository
	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

type CreateSlideBlockParams struct {
	Position  uint
	BlockType string
	Content   map[string]any
}

type CreateSlideParams struct {
	Position uint
	Duration uint // В миллисекундах
	Blocks   []CreateSlideBlockParams
}

type CreateAttractionParams struct {
	Slug     string
	Name     string
	City     string
	Category string
	Publish  bool
	Geom     types.Point
	Slides   []CreateSlideParams
}

func (s *Service) Create(ctx context.Context, params CreateAttractionParams) (Model, error) {
	s.logger.Info("start Create", zap.String("layer", "domain service"), zap.String("slug", params.Slug))

	if err := validateCreate(params); err != nil {
		return Model{}, err
	}

	obj := buildModel(params)
	// Слайды и блоки GORM пишет ассоциациями в той же транзакции, что и саму достопримечательность
	if err := s.repo.Create(ctx, &obj); err != nil {
		return Model{}, err
	}
	return obj, nil
}

func buildModel(params CreateAttractionParams) Model {
	status := StatusDraft
	if params.Publish {
		status = StatusPublished
	}

	return Model{
		Slug:     params.Slug,
		Name:     params.Name,
		City:     params.City,
		Category: params.Category,
		Status:   status,
		Geom:     params.Geom,
		Slides:   buildSlides(params.Slides),
	}
}

func buildSlides(params []CreateSlideParams) []Slide {
	slides := make([]Slide, 0, len(params))
	for _, sp := range params {
		blocks := make([]SlideBlock, 0, len(sp.Blocks))
		for _, bp := range sp.Blocks {
			blocks = append(blocks, SlideBlock{
				Position:  bp.Position,
				BlockType: BlockType(bp.BlockType),
				Content:   bp.Content,
			})
		}
		slides = append(slides, Slide{
			Position:   sp.Position,
			DurationMs: sp.Duration,
			Blocks:     blocks,
		})
	}
	return slides
}
