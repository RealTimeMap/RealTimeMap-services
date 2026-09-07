package group

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

type Repository interface {
	Create(ctx context.Context, obj *Model) error
	Update(ctx context.Context, obj *Model) error
	List(ctx context.Context, userID uint, params pagination.Params) ([]*Model, int64, error)
	Delete(ctx context.Context, obj *Model) error
	GetBatch(ctx context.Context, userID uint, ids []uint) ([]*Model, error)
}
