package model

import "gorm.io/gorm"

type ReactionType string

const (
	Like    ReactionType = "like"
	Dislike ReactionType = "dislike"
)

type Reaction struct {
	gorm.Model

	CommentID uint `gorm:"uniqueIndex:idx_user_comment"`
	UserID    uint `gorm:"uniqueIndex:idx_user_comment"`

	Type ReactionType
}
