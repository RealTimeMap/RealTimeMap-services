package model

import "time"

type BlockedUser struct {
	UserID        uint `gorm:"primaryKey;autoIncrement:false"`
	BlockedUserID uint `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt     time.Time
}
