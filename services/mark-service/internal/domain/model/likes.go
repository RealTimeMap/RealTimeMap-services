package model

type MarkReaction struct {
	ID     uint `gorm:"primaryKey"`
	MarkID uint `gorm:"uniqueIndex:idx_mark_user;not null"`
	UserID uint `gorm:"uniqueIndex:idx_mark_user;not null"`
}
