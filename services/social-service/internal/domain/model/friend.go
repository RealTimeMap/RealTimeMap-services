package model

import "time"

type FriendshipStatus string

const (
	Waiting  FriendshipStatus = "waiting"
	Accepted FriendshipStatus = "accepted"
)

type Friendship struct {
	UserID    uint `gorm:"primaryKey;autoIncrement:false"`
	FriendID  uint `gorm:"primaryKey;autoIncrement:false"`
	Status    FriendshipStatus
	CreatedAt time.Time
}
