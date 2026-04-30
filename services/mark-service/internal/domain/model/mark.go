package model

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"gorm.io/gorm"
)

var AllowedDuration = []int{12, 24, 36, 48}

type Mark struct {
	gorm.Model
	ID             int `gorm:"primarykey"`
	MarkName       string
	UserID         int
	UserName       string `gorm:"not null"`
	CategoryID     int
	Category       Category
	AdditionalInfo *string
	StartAt        time.Time `gorm:"index:idx_marks_time,priority:1,where:NOT is_ended"`
	EndAt          time.Time `gorm:"index:idx_marks_time,priority:2"`
	IsEnded        bool      `gorm:"default:false"`
	//Duration       int          `gorm:"default:12"`
	Geom    types.Point  `gorm:"type:geometry(POINT,4326);not null"`
	Geohash string       `gorm:"not null"`
	Photos  types.Photos `gorm:"type:jsonb"`

	Owner *UserProfile `gorm:"-" json:"-"`
}

func (m *Mark) BeforeCreate(_ *gorm.DB) (err error) {
	// m.EndAt = m.StartAt.Add(time.Duration(m.Duration) * time.Hour)
	return
}

// DefaultEndAt метод добавляет время окончания для тех меток где не указано EndAt (Быстрые метки)
func (m *Mark) DefaultEndAt() {
	m.EndAt = m.StartAt.Add(time.Duration(1) * time.Hour)
	return
}

type Cluster struct {
	Center types.Point
	Count  int
}
