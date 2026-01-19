package model

import (
	"time"
)

// UserProgress хранит прогресс пользователя в системе геймификации.
// CurrentXP - это total XP (накопленный с начала), не XP на текущем уровне.
type UserProgress struct {
	UserID       uint `gorm:"primaryKey;autoIncrement:false"`
	CurrentLevel uint `gorm:"not null;default:1;index"`
	CurrentXP    uint `gorm:"not null;default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

	Level Level `gorm:"foreignKey:CurrentLevel;references:Level"`
}
