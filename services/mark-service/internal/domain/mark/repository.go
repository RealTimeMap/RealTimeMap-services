package mark

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

type Repository interface {
	Create(ctx context.Context, data *Mark) (*Mark, error)
	TodayCreated(ctx context.Context, userID uint) (int64, error)
	GetMarksInArea(ctx context.Context, filter Filter) ([]*Mark, error)
	GetUserMarks(ctx context.Context, userID uint, params pagination.Params) ([]*Mark, int64, error)
	GetMarksInCluster(ctx context.Context, filter Filter) ([]*Cluster, error)
	Exist(ctx context.Context, id int) (bool, error)
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*Mark, error)
	Update(ctx context.Context, id uint, mark *Mark) (*Mark, error)

	// Специфические для админ панели запросы

	GetAll(ctx context.Context, params pagination.Params) ([]*Mark, int64, error)
}
