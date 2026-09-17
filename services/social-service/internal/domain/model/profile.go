package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

type Profile struct {
	UserID    uint        `gorm:"primaryKey;autoIncrement:false"`
	Username  string      `gorm:"index"`
	Avatar    types.Photo `gorm:"type:jsonb"`
	Tag       string      `gorm:"index:idx_tag,unique"`
	IsPrivate bool        `gorm:"default:false"`

	// IsAdmin — зеркало признака администратора из auth-сервиса.
	//
	// Хранится копией, а не спрашивается по gRPC на каждый запрос: профиль
	// отдаётся в списках (поиск, участники чата, авторы комментариев), и поход
	// в auth за каждым элементом превратил бы выдачу в N+1. Источник истины
	// остаётся за auth — сюда значение приезжает событиями user.registered и
	// user.updated и никогда не меняется запросами пользователя.
	//
	// Авторизацию по этому полю строить нельзя: между событием и его
	// применением значение здесь отстаёт от auth. Проверка прав живёт на
	// gateway по заголовку X-User-Admin (pkg/middleware/auth.AdminOnly).
	IsAdmin         bool            `gorm:"default:false;index"`
	PrivacySettings PrivacySettings `gorm:"type:jsonb"`
	Gamification    Progress        `gorm:"-" json:"-"`
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
