package model

import "gorm.io/gorm"

type EntityType string

const (
	EntityMark EntityType = "mark"
)

const MaxDepth uint = 2 // Константа для обозначения максимальной глубины вложенности комментариев

type Comment struct {
	gorm.Model
	UserID  uint // ID Юзера
	Content string

	ParentID *uint     `gorm:"index"`
	Parent   *Comment  `gorm:"foreignKey:ParentID"`
	Replies  []Comment `gorm:"foreignKey:ParentID"`

	EntityType EntityType `gorm:"size:32;not null;index:idx_entity"`
	EntityID   uint       `gorm:"not null;index:idx_entity"`

	LikesCount    uint `gorm:"default:0"`
	DislikesCount uint `gorm:"default:0"`

	Depth uint `gorm:"not null; default:0"`
}
