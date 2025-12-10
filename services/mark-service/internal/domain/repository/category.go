package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
)

// CategoryRepository интерфейс для слоя репозитория
type CategoryRepository interface {
	Create(ctx context.Context, data *model.Category) (*model.Category, error)
	GetByName(ctx context.Context, name string) (*model.Category, error)
}
