package mark

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/app/use_cases/mark_action"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/category"
)

// Coordinates represents coordinates response
// @name Coordinates
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

type OwnerResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Tag      string `json:"tag"`
}

func NewOwnerResponse(u mark_action.UserResult) OwnerResponse {
	return OwnerResponse{
		ID:       u.ID,
		Username: u.Username,
		Tag:      u.Tag,
		Avatar:   u.Avatar,
	}
}

// ResponseMark represents mark_action response
// @name MarkResponse
type ResponseMark struct {
	ID       int          `json:"id"`
	MarKName string       `json:"markName"`
	Geom     *Coordinates `json:"geom"`
	Photos   []string     `json:"photos"`
}

func NewResponseMark(data mark_action.MarkResult) ResponseMark {
	response := ResponseMark{
		ID:       int(data.ID),
		MarKName: data.MarkName,
		Geom:     NewFromPoint(data.Geom),
		Photos:   data.Photos,
	}
	return response
}

func NewMultipleResponseMarkV2(data []mark_action.MarkResult) []ResponseMark {
	response := make([]ResponseMark, len(data))
	for i := range response {
		response[i] = NewResponseMark(data[i])
	}
	return response
}

// ResponseCluster represents cluster of marks response
// @name ResponseCluster
type ResponseCluster struct {
	Center *Coordinates `json:"center"`
	Count  int          `json:"count"`
}

func NewResponseCluster(data mark_action.ClusterResult) ResponseCluster {
	response := ResponseCluster{
		Center: NewFromPoint(data.Center),
		Count:  data.Count,
	}
	return response
}

func NewMultipleResponseCluster(data []mark_action.ClusterResult) []ResponseCluster {
	response := make([]ResponseCluster, len(data))
	for i := range response {
		response[i] = NewResponseCluster(data[i])
	}
	return response
}

type Date struct {
	StartAt         time.Time `json:"startAt"`
	EndAt           time.Time `json:"endAt"`
	ProgressPercent float64   `json:"progressPercent"`
	DaysPassed      int       `json:"daysPassed"`
	DaysLeft        int       `json:"daysLeft"`
}

func NewDate(m mark_action.DateResult) Date {
	return Date{
		StartAt:         m.StartAt,
		EndAt:           m.EndAt,
		ProgressPercent: m.ProgressPercent,
		DaysLeft:        m.DaysLeft,
		DaysPassed:      m.DaysPassed,
	}
}

type Meta struct {
	Status   string `json:"status"`
	MarkType string `json:"markType"`
}

func NewMeta(m mark_action.MetaResult) Meta {
	return Meta{
		Status:   m.Status,
		MarkType: m.MarkType,
	}
}

type DetailMarkResponse struct {
	ID             uint                      `json:"id"`
	MarKName       string                    `json:"markName"`
	AdditionalInfo *string                   `json:"additionalInfo,omitempty"`
	Category       category.ResponseCategory `json:"category"`
	Geom           *Coordinates              `json:"geom"`
	User           OwnerResponse             `json:"owner"`
	Photos         []string                  `json:"photos"`
	Date           Date                      `json:"date"`
	Meta           Meta                      `json:"meta"`
}

func NewDetailMarkResponse(data mark_action.DetailMarkResult) DetailMarkResponse {
	date := NewDate(data.Date)
	response := DetailMarkResponse{
		ID:             data.ID,
		MarKName:       data.MarkName,
		AdditionalInfo: data.AdditionalInfo,
		Geom:           NewFromPoint(data.Geom),
		User:           NewOwnerResponse(data.Owner),
		Date:           date,
		Meta:           NewMeta(data.Meta),
		Photos:         data.Photos,
	}
	if data.Category.ID != 0 {
		response.Category = category.NewResponseCategoryMark(data.Category)
	}

	return response
}
