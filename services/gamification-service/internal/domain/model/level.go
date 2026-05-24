package model

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
)

// Level представляет уровень в системе геймификации.
// XP накопительный - каждый уровень требует total XP от начала.
type Level struct {
	Level      uint   `gorm:"primaryKey;autoIncrement:false"`
	Title      string `gorm:"type:varchar(100)"`
	XPRequired uint   `gorm:"not null;check:xp_required >= 0"`
	CreatedAt  time.Time
}

// Percent вычисляет процент прогресса достяжения до достяжения нового уровня
func (l *Level) Percent(xp float64) float64 {
	p := utils.Percent(int64(xp), int64(l.XPRequired))
	if p > 100 {
		p = 100
	}
	return p
}
