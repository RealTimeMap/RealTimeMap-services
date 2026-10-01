package attraction

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

// UpdateAttractionParams — частичное обновление: nil означает "не менять".
// Slides заменяются целиком; пустой срез удаляет все слайды.
type UpdateAttractionParams struct {
	ID       uuid.UUID
	Slug     *string
	Name     *string
	City     *string
	Category *string
	Status   *string
	Geom     *types.Point
	Slides   *[]CreateSlideParams
}

func (s *Service) Update(ctx context.Context, params UpdateAttractionParams) (Model, error) {
	s.logger.Info("start Update", zap.String("layer", "domain service"), zap.String("id", params.ID.String()))

	if err := validateUpdate(params); err != nil {
		return Model{}, err
	}

	obj, err := s.repo.GetByID(ctx, params.ID)
	if err != nil {
		return Model{}, err
	}

	if params.Slug != nil {
		obj.Slug = *params.Slug
	}
	if params.Name != nil {
		obj.Name = *params.Name
	}
	if params.City != nil {
		obj.City = *params.City
	}
	if params.Category != nil {
		obj.Category = *params.Category
	}
	if params.Status != nil {
		obj.Status = Status(*params.Status)
	}
	if params.Geom != nil {
		obj.Geom = *params.Geom
	}
	replaceSlides := params.Slides != nil
	if replaceSlides {
		obj.Slides = buildSlides(*params.Slides)
	}

	if err := s.repo.Update(ctx, &obj, replaceSlides); err != nil {
		return Model{}, err
	}
	return obj, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	s.logger.Info("start Delete", zap.String("layer", "domain service"), zap.String("id", id.String()))
	return s.repo.Delete(ctx, id)
}
