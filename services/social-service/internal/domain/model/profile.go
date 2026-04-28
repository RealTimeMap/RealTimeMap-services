package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

type Profile struct {
	UserID          uint            `gorm:"primaryKey;autoIncrement:false"`
	Username        string          `gorm:"index"`
	Avatar          types.Photo     `gorm:"type:jsonb"`
	Tag             string          `gorm:"index:idx_tag,unique"`
	IsPrivate       bool            `gorm:"default:false"`
	PrivacySettings PrivacySettings `gorm:"type:jsonb"`
}

type PrivacySettings struct {
	ShowInSearch bool `json:"showInSearch"`
}

func (p *PrivacySettings) Scan(val interface{}) error {
	if val == nil {
		*p = PrivacySettings{}
		return nil
	}

	var data []byte
	switch v := val.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into PrivacySettings", val)
	}

	return json.Unmarshal(data, p)
}

func (p PrivacySettings) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// DefaultPrivacySettings дефолтные настройки профиля пользователя
func DefaultPrivacySettings() PrivacySettings {
	return PrivacySettings{
		ShowInSearch: true,
	}
}
