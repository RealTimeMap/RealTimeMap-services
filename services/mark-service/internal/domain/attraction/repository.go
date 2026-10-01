package attraction

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, obj *Model) error
	// ListByCity отдаёт опубликованные достопримечательности города без слайдов
	ListByCity(ctx context.Context, city string) ([]Model, error)
	// GetPublished отдаёт опубликованную достопримечательность со слайдами и блоками
	GetPublished(ctx context.Context, id uuid.UUID) (Model, error)
	// GetByID отдаёт достопримечательность в любом статусе со слайдами и блоками
	GetByID(ctx context.Context, id uuid.UUID) (Model, error)
	// Update сохраняет поля; при replaceSlides слайды пересоздаются в той же транзакции
	Update(ctx context.Context, obj *Model, replaceSlides bool) error
	Delete(ctx context.Context, id uuid.UUID) error
}
