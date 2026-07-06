package category

import (
	"context"
)

// Repository интерфейс для слоя репозитория
type Repository interface {
	Create(ctx context.Context, data *Category) (*Category, error)
	GetByName(ctx context.Context, name string) (*Category, error)
	GetByID(ctx context.Context, id uint) (*Category, error)
	Exist(ctx context.Context, id int) (bool, error)
	GetAll(ctx context.Context) ([]*Category, error)
}
