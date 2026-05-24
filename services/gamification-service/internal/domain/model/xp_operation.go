package model

import (
	"gorm.io/gorm"
)

type HistoryStatus string

const (
	Credited HistoryStatus = "credited"
	Reverted HistoryStatus = "reverted"
)

type SourceType string

const (
	SourceEvent       SourceType = "event"
	SourceAchievement SourceType = "achievement"
	SourceManual      SourceType = "manual"
)

type XPOperation struct {
	gorm.Model

	UserID uint
	Amount int

	SourceType SourceType `gorm:"type:varchar(32);index:idx_source"`
	SourceID   uint       `gorm:"index:idx_source"`

	Status HistoryStatus `gorm:"type:varchar(16);default:'credited'"`
	Reason *string       `gorm:"type:varchar(256)"`
}
