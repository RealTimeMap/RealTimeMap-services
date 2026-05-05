package mark

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
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

func NewOwnerResponse(u *model.UserProfile) OwnerResponse {
	if u == nil {
		return OwnerResponse{}
	}
	return OwnerResponse{
		ID:       u.ID,
		Username: u.Username,
		Tag:      u.Tag,
		Avatar:   u.Avatar,
	}
}

// ResponseMark represents mark response
// @name MarkResponse
type ResponseMark struct {
	ID       int          `json:"id"`
	MarKName string       `json:"markName"`
	Geom     *Coordinates `json:"geom"`
	Photos   []string     `json:"photos"`
}

func NewResponseMark(data *model.Mark) *ResponseMark {
	response := &ResponseMark{
		ID:       data.ID,
		MarKName: data.MarkName,
		Geom:     NewFromPoint(data.Geom),
	}
	for _, photo := range data.Photos {
		response.Photos = append(response.Photos, photo.URL)
	}
	return response
}

func NewMultipleResponseMark(data []*model.Mark) []*ResponseMark {
	response := make([]*ResponseMark, len(data))
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

func NewResponseCluster(data *model.Cluster) *ResponseCluster {
	response := &ResponseCluster{
		Center: NewFromPoint(data.Center),
		Count:  data.Count,
	}
	return response
}

func NewMultipleResponseCluster(data []*model.Cluster) []*ResponseCluster {
	response := make([]*ResponseCluster, len(data))
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

func NewDate(m *model.Mark) Date {
	return Date{
		StartAt:         m.StartAt,
		EndAt:           m.EndAt,
		ProgressPercent: m.ProgressPercent(),
		DaysLeft:        m.DaysLeft(),
		DaysPassed:      m.DaysSinceStart(),
	}
}

type Meta struct {
	Status   string `json:"status"`
	MarkType string `json:"markType"`
}

func NewMeta(m *model.Mark) Meta {
	return Meta{
		Status:   m.Status(),
		MarkType: string(m.GetMarkType()),
	}
}

type DetailMarkResponse struct {
	ID             int                        `json:"id"`
	MarKName       string                     `json:"markName"`
	AdditionalInfo *string                    `json:"additionalInfo,omitempty"`
	Category       *category.ResponseCategory `json:"category"`
	Geom           *Coordinates               `json:"geom"`
	User           OwnerResponse              `json:"owner"`
	Photos         []string                   `json:"photos"`
	Date           Date                       `json:"date"`
	Meta           Meta                       `json:"meta"`
}

func NewDetailMarkResponse(data *model.Mark) DetailMarkResponse {
	date := NewDate(data)
	response := DetailMarkResponse{
		ID:             data.ID,
		MarKName:       data.MarkName,
		AdditionalInfo: data.AdditionalInfo,
		Geom:           NewFromPoint(data.Geom),
		User:           NewOwnerResponse(data.Owner),
		Date:           date,
		Meta:           NewMeta(data),
	}
	if data.Category.ID != 0 {
		response.Category = category.NewResponseCategory(&data.Category)
	}
	for _, photo := range data.Photos {
		response.Photos = append(response.Photos, photo.URL)
	}
	return response
}
