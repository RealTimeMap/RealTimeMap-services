package model

import "time"

// Subscription односторонняя подписка одного пользователя на профиль другого.
// В отличие от Friendship связь асимметрична: подписка SubscriberID -> TargetID
// не требует подтверждения и не создаёт обратной связи. Взаимная подписка
// двух пользователей друг на друга дружбой не является.
type Subscription struct {
	SubscriberID uint `gorm:"primaryKey;autoIncrement:false;index"`
	TargetID     uint `gorm:"primaryKey;autoIncrement:false;index"`
	CreatedAt    time.Time
}
