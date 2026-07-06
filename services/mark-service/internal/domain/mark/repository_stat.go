package mark

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
)

type StatsRepository interface {
	// GetMarkCount получение общего количества меток пользователя
	GetMarkCount(ctx context.Context, userID uint) (int64, error)
	// GetCountForMonths метод получен счетчика по месяцам в течение года
	GetCountForMonths(ctx context.Context, userID uint, year int) ([]MonthlyActivity, error)
	// GetCountPerPeriod метод получения счетчика по дням в течении определенного периода
	GetCountPerPeriod(ctx context.Context, userID uint, start, end time.Time) ([]DayActivity, error)
	// GetPopularCategories Получение популярных категорий пользователя на основе меток
	GetPopularCategories(ctx context.Context, userID uint) ([]category.CategoryStat, error)
}
