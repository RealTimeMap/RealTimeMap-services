package repository

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/valueobject"
)

type Filter struct {
	BoundingBox valueobject.BoundingBox
	ZoomLevel   int
	StartAt     time.Time
	EndAt       time.Time
	ShowEnded   bool
	Duration    int
}

type MarkRepository interface {
	Create(ctx context.Context, data *model.Mark) (*model.Mark, error)
	TodayCreated(ctx context.Context, userID int) (int64, error)
	GetMarksInArea(ctx context.Context, filter Filter) ([]*model.Mark, error) // TODO перенести фильтры куда то...
	GetMarksInCluster(ctx context.Context, filter Filter) ([]*model.Cluster, error)
	Exist(ctx context.Context, id int) (bool, error)
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.Mark, error)
	Update(ctx context.Context, id int, mark *model.Mark) (*model.Mark, error)
}
