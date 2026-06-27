package repository

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/valueobject"
)

type Filter struct {
	BoundingBox valueobject.BoundingBox
	ZoomLevel   float64
	StartAt     time.Time
	EndAt       time.Time
	ShowEnded   bool
	Duration    int
}

type MarkRepository interface {
	Create(ctx context.Context, data *model.Mark) (*model.Mark, error)
	TodayCreated(ctx context.Context, userID int) (int64, error)
	GetMarksInArea(ctx context.Context, filter Filter) ([]*model.Mark, error)
	GetUserMarks(ctx context.Context, userID uint, params pagination.Params) ([]*model.Mark, int64, error)
	GetMarksInCluster(ctx context.Context, filter Filter) ([]*model.Cluster, error)
	Exist(ctx context.Context, id int) (bool, error)
	Delete(ctx context.Context, id int) error
	GetByID(ctx context.Context, id int) (*model.Mark, error)
	Update(ctx context.Context, id int, mark *model.Mark) (*model.Mark, error)

	// Специфические для админ панели запросы

	GetAll(ctx context.Context, params pagination.Params) ([]*model.Mark, int64, error)
}

type MarkStatsRepository interface {
	// GetMarkCount получение общего количества меток пользователя
	GetMarkCount(ctx context.Context, userID uint) (int64, error)
	// GetCountForMonths метод получен счетчика по месяцам в течение года
	GetCountForMonths(ctx context.Context, userID uint, year int) ([]model.MonthlyActivity, error)
	// GetCountPerPeriod метод получения счетчика по дням в течении определенного периода
	GetCountPerPeriod(ctx context.Context, userID uint, start, end time.Time) ([]model.DayActivity, error)
	// GetPopularCategories Получение популярных категорий пользователя на основе меток
	GetPopularCategories(ctx context.Context, userID uint) ([]model.CategoryStat, error)
}
