package model

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"gorm.io/gorm"
)

var AllowedDuration = [4]int{12, 24, 36, 48}

type Mark struct {
	gorm.Model
	MarkName       string
	UserID         int
	UserName       string
	CategoryID     int
	CategoryName   string
	AdditionalInfo *string
	StartAt        time.Time
	IsEnded        bool         `gorm:"default:false"`
	Duration       int          `gorm:"default:12"`
	Geom           types.Point  `gorm:"type:geometry(POINT,4326);not null"`
	Photos         types.Photos `gorm:"type:jsonb"`
}

// EndAt вычисляемое время окончание действия метки
func (m *Mark) EndAt() time.Time {
	return m.StartAt.Add(time.Duration(m.Duration) * time.Hour)
}
