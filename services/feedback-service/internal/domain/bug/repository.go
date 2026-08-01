package bug

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

type Filter struct {
	Pagination pagination.Params
	Status     *Status
	Tag        *Tag
}

type Repository interface {
	Create(ctx context.Context, data *Model) error
	GetByID(ctx context.Context, id uint) (*Model, error)
	GetList(ctx context.Context, filter Filter) ([]Model, error)
}
