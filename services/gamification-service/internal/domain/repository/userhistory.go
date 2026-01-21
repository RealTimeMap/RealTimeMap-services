package repository

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type UserExpHistoryRepository interface {
	Create(ctx context.Context, userExpHistory *model.UserExpHistory) (*model.UserExpHistory, error)
	CountForDay(ctx context.Context, userID, configID uint, date time.Time) (int64, error)
	ExistsBySourceID(ctx context.Context, userID, configID, sourceID uint) (bool, error)
}
