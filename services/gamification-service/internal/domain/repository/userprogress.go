package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type UserProgressRepository interface {
	Update(ctx context.Context, user *model.UserProgress) (*model.UserProgress, error)
	Create(ctx context.Context, user *model.UserProgress) (*model.UserProgress, error)
	GetOrCreate(ctx context.Context, userID uint) (*model.UserProgress, error)
	GetByID(ctx context.Context, userID uint) (*model.UserProgress, error)
	GetTopUsers(ctx context.Context) ([]*model.UserProgress, error)
}
