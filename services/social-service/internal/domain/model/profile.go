package model

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/types"

type Profile struct {
	UserID          uint            `gorm:"primaryKey;autoIncrement:false"`
	Avatar          types.Photo     `gorm:"type:jsonb"`
	PrivacySettings PrivacySettings `gorm:"type:jsonb"`
}

type PrivacySettings struct {
	ShowInSearch bool `json:"showInSearch"`
}
