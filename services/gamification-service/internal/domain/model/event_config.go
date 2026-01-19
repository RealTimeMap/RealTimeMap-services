package model

import "gorm.io/gorm"

type EventConfig struct {
	gorm.Model
	EventType      string
	KafkaEventType string
	Description    *string `gorm:"type:varchar(256)"`
	RewardExp      uint
	IsActive       bool `gorm:"default:true"`
	IsRepeatable   bool `gorm:"default:true"`
	DailyLimit     *uint
}
