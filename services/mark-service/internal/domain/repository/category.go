package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
)

// CategoryRepository интерфейс для слоя репозитория
type CategoryRepository interface {
	Create(ctx context.Context, data *model.Category) (*model.Category, error)
	GetByName(ctx context.Context, name string) (*model.Category, error)
	GetByID(ctx context.Context, id int) (*model.Category, error)
	Exist(ctx context.Context, id int) (bool, error)
	GetAll(ctx context.Context) ([]*model.Category, error)
}
