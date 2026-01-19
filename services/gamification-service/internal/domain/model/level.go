package model

import (
	"time"
)

// Level представляет уровень в системе геймификации.
// XP накопительный - каждый уровень требует total XP от начала.
type Level struct {
	Level      uint   `gorm:"primaryKey;autoIncrement:false"`
	Title      string `gorm:"type:varchar(100)"`
	XPRequired uint   `gorm:"not null;check:xp_required >= 0"`
	CreatedAt  time.Time
}
