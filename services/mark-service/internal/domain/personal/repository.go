package personal

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

type Repository interface {
	Create(ctx context.Context, obj *Model) error
	List(ctx context.Context, userID uint, since *uint, upTo uint, limit int) ([]Model, error)
}

type RevisionRepository interface {
	UpsertRevision(ctx context.Context, userID uint) (uint, error)
	GetRevision(ctx context.Context, userID uint) (uint, error)
}

type GroupRepository interface {
	Create(ctx context.Context, obj *Group) error
	Update(ctx context.Context, obj *Group) error
	List(ctx context.Context, userID uint, params pagination.Params) ([]*Group, int64, error)
	Delete(ctx context.Context, obj *Group) error
	GetBatch(ctx context.Context, userID uint, ids []uint) ([]*Group, error)
	Get(ctx context.Context, userID uint, since *uint, upTo uint, limit int) ([]Group, error)
}
