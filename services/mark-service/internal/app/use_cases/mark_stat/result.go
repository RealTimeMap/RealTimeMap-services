package mark_stat

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
)

type CategoryStatResult struct {
	CategoryName string
	Count        int64
	Percent      float64
}

func toCategoryStatResult(obj category.CategoryStat) CategoryStatResult {
	return CategoryStatResult{
		CategoryName: obj.CategoryName,
		Count:        obj.Count,
		Percent:      obj.Percent,
	}
}

func toMultiCategoryStatResult(obj []category.CategoryStat) []CategoryStatResult {
	result := make([]CategoryStatResult, 0, len(obj))
	for _, i := range obj {
		result = append(result, toCategoryStatResult(i))
	}
	return result
}

type MonthlyActivityResult struct {
	Month string
	Count int64
}

func toMonthlyActivityResult(obj mark.MonthlyActivity) MonthlyActivityResult {
	return MonthlyActivityResult{
		Month: obj.Month,
		Count: obj.Count,
	}
}

func toMultiMonthlyActivityResult(obj []mark.MonthlyActivity) []MonthlyActivityResult {
	result := make([]MonthlyActivityResult, 0, len(obj))
	for _, i := range obj {
		result = append(result, toMonthlyActivityResult(i))
	}
	return result
}

type DayActivityResult struct {
	Day   time.Time
	Count int64
}

func toDayActivityResult(obj mark.DayActivity) DayActivityResult {
	return DayActivityResult{
		Day:   obj.Day,
		Count: obj.Count,
	}
}
func toMultiDayActivityResult(obj []mark.DayActivity) []DayActivityResult {
	result := make([]DayActivityResult, 0, len(obj))
	for _, i := range obj {
		result = append(result, toDayActivityResult(i))
	}
	return result
}
