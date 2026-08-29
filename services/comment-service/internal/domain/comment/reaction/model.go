package reaction

import "time"

type Reaction struct {
	ID        uint `gorm:"primaryKey"`
	CommentID uint `gorm:"uniqueIndex:idx_user_comment;not null"`
	UserID    uint `gorm:"uniqueIndex:idx_user_comment;not null"`
	CreatedAt time.Time
}
