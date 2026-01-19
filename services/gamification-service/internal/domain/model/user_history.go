package model

import (
	"time"

	"gorm.io/gorm"
)

type HistoryStatus string

const (
	Credited HistoryStatus = "credited"
	Reverted HistoryStatus = "reverted"
)

type UserExpHistory struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index:idx_daily_limit,priority:3"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	UserID    uint
	EarnedExp uint
	Status    HistoryStatus
	SoursID   *uint
	ConfigID  uint
	Config    EventConfig `gorm:"foreignKey:ConfigID"`
}
