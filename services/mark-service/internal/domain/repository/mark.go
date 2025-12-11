package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
)

type MarkRepository interface {
	Create(ctx context.Context, data *model.Mark) (*model.Mark, error)
}
