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
	MarKName       string                     `json:"markName"`
	AdditionalInfo *string                    `json:"additionalInfo,omitempty"`
	Category       *category.ResponseCategory `json:"category"`
	Geom           *Coordinates               `json:"geom"`
}

func NewResponseMark(data *model.Mark) *ResponseMark {
	response := &ResponseMark{
		ID:             data.ID,
		MarKName:       data.MarkName,
		AdditionalInfo: data.AdditionalInfo,
		Geom:           NewFromPoint(data.Geom),
	}
	if data.Category.ID != 0 {
		response.Category = category.NewResponseCategory(&data.Category)
	}
	return response
}

func NewMultiplyResponseMark(data []*model.Mark) []*ResponseMark {
	response := make([]*ResponseMark, len(data))
	for i := range response {
		response[i] = NewResponseMark(data[i])
	}
	return response
}
