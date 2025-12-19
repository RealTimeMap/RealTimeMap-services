package mark

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/valueobject"
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
	ZoomLevel int       `json:"zoomLevel" binding:"-"`
	StartAt   time.Time `json:"startAt" binding:"required"`
	EndAt     time.Time `json:"endAt" binding:"-"`
}

func ToInputFilter(data FilterParams) repository.Filter {
	return repository.Filter{BoundingBox: valueobject.BoundingBox{
		LeftTop: valueobject.Point{
			Lon: data.Screen.LeftTop.Longitude,
			Lat: data.Screen.LeftTop.Latitude,
		},
		RightBottom: valueobject.Point{
			Lon: data.Screen.RightBottom.Longitude,
			Lat: data.Screen.RightBottom.Latitude,
		},
	},
		ZoomLevel: data.ZoomLevel,
		StartAt:   data.StartAt,
		EndAt:     data.EndAt,
	}
}
