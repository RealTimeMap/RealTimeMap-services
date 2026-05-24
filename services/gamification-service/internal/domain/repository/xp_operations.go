package repository

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type XPOperationRepository interface {
	Create(ctx context.Context, XPOperation *model.XPOperation) (*model.XPOperation, error)
	GetCountEventsForDay(ctx context.Context, userID uint, date time.Time) (int64, error)
}
