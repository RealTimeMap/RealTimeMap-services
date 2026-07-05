package accrual

import (
	"time"

	"gorm.io/gorm"
)

type MarkReaction struct {
	gorm.Model
	ID        uint `gorm:"primaryKey"`
	MarkID    uint `gorm:"uniqueIndex:idx_mark_user;not null"`
	UserID    uint `gorm:"uniqueIndex:idx_mark_user;not null"`
	CreatedAt time.Time
}
