package token

import "context"

type Repository interface {
	Create(ctx context.Context, obj *Model) error
	GetByUser(ctx context.Context, userID uint) ([]Model, error)
}
