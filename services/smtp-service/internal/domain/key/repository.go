package key

import "context"

type Repository interface {
	Get(ctx context.Context, key string) (*Model, error)
	Update(ctx context.Context, obj *Model) error
	Create(ctx context.Context, obj *Model) error
}
