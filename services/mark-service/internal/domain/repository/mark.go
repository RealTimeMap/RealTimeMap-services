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

func (f Filter) GeoHashes() []string {
	return f.BoundingBox.GeoHashes()
}

type MarkRepository interface {
	Create(ctx context.Context, data *model.Mark) (*model.Mark, error)
	TodayCreated(ctx context.Context, userID int) (int64, error)
	GetMarksInArea(ctx context.Context, filter Filter) ([]*model.Mark, error) // TODO перенести фильтры куда то...
	GetMarksInCluster(ctx context.Context, filter Filter) ([]*model.Cluster, error)
}
