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

func NewMultiResponseCategory(data []*model.Category) []*ResponseCategory {
	response := make([]*ResponseCategory, len(data))
	for i, c := range data {
		response[i] = NewResponseCategory(c)
	}
	return response
}

type ResponseCreateData struct {
	Categories []*ResponseCategory `json:"allowedCategories"`
	Duration   []int               `json:"allowedDuration"`
}

func NewResponseCreateData(categories []*model.Category, allowedDuration []int) *ResponseCreateData {
	categoryResponse := NewMultiResponseCategory(categories)
	return &ResponseCreateData{
		Categories: categoryResponse,
		Duration:   allowedDuration,
	}
}
