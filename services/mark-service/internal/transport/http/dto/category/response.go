package category

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/category"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_action"
)

type ResponseCategory struct {
	ID           uint   `json:"id"`
	CategoryName string `json:"categoryName"`
	Color        string `json:"color"`
	Icon         string `json:"icon"`
}

func NewResponseCategoryMark(data mark_action.CategoryResult) ResponseCategory {
	return ResponseCategory{
		ID:           data.ID,
		CategoryName: data.CategoryName,
		Color:        data.Color,
		Icon:         data.Icon,
	}

}
func NewResponseCategory(data category.CategoryResult) ResponseCategory {
	return ResponseCategory{
		ID:           data.ID,
		CategoryName: data.CategoryName,
		Color:        data.Color,
		Icon:         data.Icon,
	}

}

func NewMultiResponseCategory(data []category.CategoryResult) []ResponseCategory {
	res := make([]ResponseCategory, 0, len(data))
	for _, obj := range data {
		res = append(res, NewResponseCategory(obj))
	}
	return res
}
