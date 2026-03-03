package model

import "time"

type FriendshipStatus string

const (
	waiting  FriendshipStatus = "waiting"
	accepted FriendshipStatus = "accepted"
)

type Friendship struct {
	UserID    uint `gorm:"primaryKey;autoIncrement:false"`
	FriendID  uint `gorm:"primaryKey;autoIncrement:false"`
	Status    FriendshipStatus
	CreatedAt time.Time
}
