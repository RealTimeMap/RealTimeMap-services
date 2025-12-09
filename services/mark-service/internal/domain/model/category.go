package model

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/types"

type Category struct {
	ID           int         `gorm:"primaryKey, autoIncrementIncrement"`
	CategoryName string      `gorm:"unique,not null"`
	Color        string      `gorm:"not null"`
	IsActive     bool        `gorm:"default:true"`
	Icon         types.Photo `gorm:"type:jsonb"`
}
