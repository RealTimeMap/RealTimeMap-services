package model

import "gorm.io/gorm"

type EventRule struct {
	gorm.Model
	EventType      string  `gorm:"uniqueIndex;type:varchar(64)"`
	KafkaEventType string  `gorm:"type:varchar(64)"`
	Description    *string `gorm:"type:varchar(256)"`

	RewardID uint     `gorm:"index"`
	Reward   XPReward `gorm:"foreignKey:RewardID"`

	IsActive     bool `gorm:"default:true"`
	IsRepeatable bool `gorm:"default:true"`
	DailyLimit   *uint
}

type XPReward struct {
	gorm.Model
	Code        string `gorm:"uniqueIndex;type:varchar(64)"`
	Amount      uint
	Description *string `gorm:"type:varchar(256)"`
}
