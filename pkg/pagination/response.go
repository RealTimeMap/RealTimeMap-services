package pagination

import "math"

// Response - универсальный ответ с пагинацией
type Response[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	Total      int64 `json:"total"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewResponse создает новый ответ с пагинацией
func NewResponse[T any](items []T, params Params, total int64) Response[T] {
	totalPages := 0
	if params.PageSize > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(params.PageSize)))
	}

	return Response[T]{
		Items:      items,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		Total:      total,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}
