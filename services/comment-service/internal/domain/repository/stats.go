package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
)

type StatisticRepository interface {
	GetCountsByPeriod(ctx context.Context, userID uint, params date.Resolved) (int64, int64, error)
	GetAllUsersComments(ctx context.Context, userID uint) (int64, error)
}
