package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type LevelRepository interface {
	Create(ctx context.Context, level *model.Level) (*model.Level, error)
	GetByLevel(ctx context.Context, level uint) (*model.Level, error)
}
