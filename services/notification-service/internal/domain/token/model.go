package token

import "time"

type Model struct {
	ID         uint   `gorm:"primaryKey"`
	UserID     uint   `gorm:"uniqueIndex:idx_user_token"`
	Token      string `gorm:"uniqueIndex:idx_user_token"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastUsedAt *time.Time
}

func (Model) TableName() string {
	return "user_notification_tokens"
}
