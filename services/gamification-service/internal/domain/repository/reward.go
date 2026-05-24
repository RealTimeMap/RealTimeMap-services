package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type XPRewardRepository interface {
	GetByID(ctx context.Context, id uint) (*model.XPReward, error)
}
