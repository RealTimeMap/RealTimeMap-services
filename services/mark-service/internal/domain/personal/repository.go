package personal

import "context"

type Repository interface {
	Create(ctx context.Context, obj *Model) error
}
