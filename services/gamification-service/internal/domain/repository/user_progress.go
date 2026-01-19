package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type UserProgressRepository interface {
	//Create(ctx context.Context, user *model.UserProgress) (*model.UserProgress, error)
	Update(ctx context.Context, user *model.UserProgress) (*model.UserProgress, error)
	GetByID(ctx context.Context, userID uint) (*model.UserProgress, error)
}
