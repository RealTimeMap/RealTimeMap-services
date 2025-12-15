package category

import "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"

type ResponseCategory struct {
	ID           int    `json:"id"`
	CategoryName string `json:"categoryName"`
	Color        string `json:"color"`
	Icon         string `json:"icon"`
}

func NewResponseCategory(data *model.Category) *ResponseCategory {
	return &ResponseCategory{
		ID:           data.ID,
		CategoryName: data.CategoryName,
		Color:        data.Color,
		Icon:         data.Icon.URL,
	}

}
