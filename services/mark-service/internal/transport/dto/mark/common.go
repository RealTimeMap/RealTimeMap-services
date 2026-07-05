package mark

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
)

type Coords struct {
	Longitude float64 `json:"lon" binding:"required,longitude" validate:"required,longitude"`
	Latitude  float64 `json:"lat" binding:"required,latitude" validate:"required,latitude"`
}

type Screen struct {
	LeftTop     Coords `json:"leftTop" binding:"required" validate:"required"`
	Center      Coords `json:"center" binding:"required" validate:"required"`
	RightBottom Coords `json:"rightBottom" binding:"required" validate:"required"`
}

type FilterParams struct {
	Screen    Screen    `json:"screen" binding:"required" validate:"required"`
	ZoomLevel float64   `json:"zoomLevel" binding:"-"`
	StartAt   time.Time `json:"startAt" binding:"required"`
	EndAt     time.Time `json:"endAt" binding:"-"`
}

func ToInputFilterV2(data FilterParams) mark.Filter {
	return mark.Filter{BoundingBox: mark.BoundingBox{
		LeftTop: mark.Point{
			Lon: data.Screen.LeftTop.Longitude,
			Lat: data.Screen.LeftTop.Latitude,
		},
		RightBottom: mark.Point{
			Lon: data.Screen.RightBottom.Longitude,
			Lat: data.Screen.RightBottom.Latitude,
		},
	},
		ZoomLevel: data.ZoomLevel,
		StartAt:   data.StartAt,
		EndAt:     data.EndAt,
	}
}
