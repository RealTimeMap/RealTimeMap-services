package model

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"gorm.io/gorm"
)

type Achievement struct {
	gorm.Model
	Code  string `gorm:"unique"`
	Title string
	Desc  string

	TriggerEventType string `gorm:"type:varchar(64)"`
	Threshold        uint   // Колво для получения

	Icon     types.Photo
	IsActive bool `gorm:"default:true"`

	RewardID uint     `gorm:"index"`
	Reward   XPReward `gorm:"foreignKey:RewardID"`

	NextID *uint        `gorm:"index"`
	Next   *Achievement `gorm:"foreignKey:NextID"`
}

type UserAchievementCount struct {
	UserID    uint   `gorm:"primaryKey;autoIncrement:false"`
	EventType string `gorm:"primaryKey;type:varchar(64)"`
	Count     uint   `gorm:"column:event_count;not null;default:0"`
	UpdatedAt time.Time
}

type UserAchievement struct {
	UserID        uint      `gorm:"primaryKey;autoincrement:false"`
	AchievementID uint      `gorm:"primaryKey;autoincrement:false"`
	UnlockedAt    time.Time `gorm:"not null"`

	Achievement Achievement `gorm:"foreignKey:AchievementID"`
}
