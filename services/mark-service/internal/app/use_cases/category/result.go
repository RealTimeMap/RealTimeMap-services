package category

import "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"

type CategoryResult struct {
	ID           uint
	CategoryName string
	Color        string
	Icon         string
}

func toCategoryResult(obj *category.Category) CategoryResult {
	return CategoryResult{
		ID:           obj.ID,
		CategoryName: obj.CategoryName,
		Color:        obj.Color,
		Icon:         obj.Icon,
	}
}

func toMultiCategoryResult(objs []*category.Category) []CategoryResult {
	results := make([]CategoryResult, 0, len(objs))
	for _, obj := range objs {
		results = append(results, toCategoryResult(obj))
	}
	return results
}
