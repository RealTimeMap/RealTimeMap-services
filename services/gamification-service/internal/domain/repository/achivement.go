package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type AchievementRepository interface {
	Create(ctx context.Context, achievement *model.Achievement) (*model.Achievement, error)
	GetByID(ctx context.Context, id uint) (*model.Achievement, error)
	GetByCode(ctx context.Context, code string) (*model.Achievement, error)
	ListUnlockableByEvent(ctx context.Context, userID uint, eventType string) ([]model.Achievement, error)
	ListNearestByUser(ctx context.Context, userID uint, limit int) ([]NearestAchievement, error)
}

type NearestAchievement struct {
	Achievement model.Achievement
	Current     uint
	Threshold   uint
}

type UserAchievementRepository interface {
	Create(ctx context.Context, userID, achID uint) error
	GetUserAchievements(ctx context.Context, userID uint, params pagination.Params) ([]*model.UserAchievement, int64, error)
}

type UserAchievementCountRepository interface {
	Create(ctx context.Context, progress *model.UserAchievementCount) (*model.UserAchievementCount, error)
	Increment(ctx context.Context, userID uint, eventType string) error
}
