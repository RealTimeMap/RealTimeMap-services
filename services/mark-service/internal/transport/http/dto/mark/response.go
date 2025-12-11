package mark

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/category"
)

type Coordinates struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

func NewFromPoint(data types.Point) *Coordinates {
	return &Coordinates{
		Type: data.GeoJSONType(),
		Coordinates: [2]float64{
			data.Lon(), data.Lat(),
		},
	}
}

type ResponseMark struct {
	ID             int                        `json:"id"`
	MarKName       string                     `json:"mark_name"`
	AdditionalInfo *string                    `json:"additional_info,omitempty"`
	CategoryID     int                        `json:"category_id,omitempty"`
	Category       *category.ResponseCategory `json:"category"`
	Geom           *Coordinates               `json:"geom"`
}

func NewResponseMark(data *model.Mark) *ResponseMark {
	response := &ResponseMark{
		ID:             data.ID,
		MarKName:       data.MarkName,
		AdditionalInfo: data.AdditionalInfo,
		CategoryID:     data.CategoryID,
		Geom:           NewFromPoint(data.Geom),
	}
	if data.Category.ID != 0 {
		response.Category = category.NewResponseCategory(&data.Category)
	}
	return response
}
