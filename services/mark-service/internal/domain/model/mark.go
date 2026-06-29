package model

import (
	"math"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"gorm.io/gorm"
)

var AllowedDuration = []int{12, 24, 36, 48}

const (
	ended      string = "Ended"
	active     string = "Active"
	notStarted string = "Not Started"
)

type MarkType string

const (
	MarkTypeTemporary MarkType = "temporary" // Временная метка - метка которая создается без времени окончания пользователя
	MarkTypeUser      MarkType = "user"
)

type Mark struct {
	gorm.Model
	ID             int `gorm:"primarykey"`
	MarkName       string
	UserID         int
	UserName       string `gorm:"not null"`
	CategoryID     int
	Category       Category
	AdditionalInfo *string

	// Временные промежутки
	StartAt time.Time `gorm:"index:idx_marks_time,priority:1,where:NOT is_ended"`
	EndAt   time.Time `gorm:"index:idx_marks_time,priority:2"`

	// Флаги для быстрой провреки
	IsTemp  bool `gorm:"default:false"`
	IsEnded bool `gorm:"default:false"`

	// Гео данные
	Geom    types.Point  `gorm:"type:geometry(POINT,4326);not null"`
	Geohash string       `gorm:"not null"`
	Photos  types.Photos `gorm:"type:jsonb"`

	// Метрики
	SharedCount int64 `gorm:"default:0"`
	LikesCount  int64 `gorm:"-"`
	IsLiked     bool  `gorm:"-"`

	Owner *UserProfile `gorm:"-" json:"-"`
}

func (m *Mark) BeforeCreate(_ *gorm.DB) (err error) {
	// m.EndAt = m.StartAt.Add(time.Duration(m.Duration) * time.Hour)
	return
}

func (m *Mark) ProgressPercent() float64 {
	now := time.Now()
	if now.Before(m.StartAt) {
		return 0.0
	}

	if now.After(m.EndAt) {
		return 100.0
	}

	totalDuration := m.EndAt.Sub(m.StartAt).Seconds()
	passedDuration := now.Sub(m.StartAt).Seconds()

	if totalDuration <= 0 {
		return 100.0
	}

	return math.Round((passedDuration/totalDuration)*10000) / 100
}

func (m *Mark) DaysLeft() int {
	now := time.Now().UTC()
	start := m.StartAt.UTC()
	end := m.EndAt.UTC()

	if now.Before(start) {
		return int(end.Sub(start).Hours() / 24) // Сколько всего дней
	}
	if now.After(end) {
		return 0
	}

	diff := end.Sub(now) // Осталось от текущего момента
	return int(diff.Hours() / 24)
}

func (m *Mark) DaysSinceStart() int {
	now := time.Now().UTC()
	start := m.StartAt.UTC()
	end := m.EndAt.UTC()

	if now.Before(start) {
		return 0
	}
	if now.After(end) {
		// Если закончилось
		return int(end.Sub(start).Hours() / 24)
	}

	diff := now.Sub(start)
	return int(diff.Hours() / 24)
}

func (m *Mark) Status() string {
	now := time.Now()

	if now.Before(m.StartAt) {
		return notStarted
	}
	if now.After(m.EndAt) {
		return ended
	}
	return active
}

// DefaultEndAt метод добавляет время окончания для тех меток где не указано EndAt (Временные метки)
func (m *Mark) DefaultEndAt() {
	m.EndAt = m.StartAt.Add(time.Duration(1) * time.Hour)
	m.IsTemp = true
	return
}

func (m *Mark) GetMarkType() MarkType {
	if m.IsTemp {
		return MarkTypeTemporary
	}
	return MarkTypeUser
}

type Cluster struct {
	Center types.Point
	Count  int
}

type MonthlyActivity struct {
	Month string
	Count int64
}

type DayActivity struct {
	Day   time.Time
	Count int64
}
