package group

import "gorm.io/gorm"

type Model struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex:idx_group"`
	Description *string
	UserID      uint `gorm:"uniqueIndex:idx_group"`
}

func (Model) TableName() string {
	return "groups"
}
