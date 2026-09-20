package token

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Model — устройство пользователя и его настройки уведомлений.
type Model struct {
	ID uint `gorm:"primaryKey"`

	UserID   uint   `gorm:"uniqueIndex:idx_user_device;index"`
	DeviceID string `gorm:"uniqueIndex:idx_user_device;size:64"`

	// Token уникален глобально, а не в паре с пользователем
	Token string `gorm:"uniqueIndex;size:512"`

	Platform Platform `gorm:"size:16"`

	// Settings живут на устройстве: выключенные уведомления на ПК не должны
	// затрагивать телефон.
	Settings Settings `gorm:"type:jsonb"`

	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastUsedAt *time.Time
}

func (Model) TableName() string {
	return "user_devices"
}

// Allows решает, доставлять ли на это устройство уведомление данного типа.
//
// Решение принимает домен, а не SQL-запрос: так оно проверяется тестом без
// базы, а в логе отправки видно, что устройство пропущено осознанно.
func (m Model) Allows(kind Kind) bool {
	if !m.Settings.Enabled {
		return false
	}
	// Пустой Kind — системная отправка (административная рассылка, тест)
	if kind == "" {
		return true
	}
	return !m.Settings.Muted[string(kind)]
}

// Platform — вид клиента. Хранится ради диагностики и будущих различий в
// формате сообщения: FCM для web требует иной payload, чем для мобильных.
type Platform string

const (
	PlatformAndroid Platform = "android"
	PlatformIOS     Platform = "ios"
	PlatformWeb     Platform = "web"
)

func (p Platform) Valid() bool {
	switch p {
	case PlatformAndroid, PlatformIOS, PlatformWeb:
		return true
	}
	return false
}

// Kind — тип уведомления, от которого можно отписаться отдельно.
type Kind string

const (
	KindChatMessage Kind = "chat"
	KindComment     Kind = "comment"
	KindSubscriber  Kind = "subscriber"
)

func (k Kind) Valid() bool {
	switch k {
	case KindChatMessage, KindComment, KindSubscriber:
		return true
	}
	return false
}

// Settings — настройки уведомлений одного устройства.
type Settings struct {
	// Enabled — общий выключатель устройства.
	Enabled bool `json:"enabled"`

	// Muted перечисляет выключенные типы. Хранится именно выключенное, а не
	// включённое: иначе новый тип уведомлений оказался бы молча выключен у
	// всех, кто зарегистрировал устройство до его появления.
	//
	// Карта, а не список: частичное обновление («выключи chat») мержится по
	// ключам, тогда как перезапись списка целиком теряла бы правки, сделанные
	// параллельно с другого устройства.
	Muted map[string]bool `json:"muted,omitempty"`
}

// DefaultSettings — настройки свежезарегистрированного устройства: пользователь
func DefaultSettings() Settings {
	return Settings{Enabled: true}
}

func (s *Settings) Scan(val interface{}) error {
	if val == nil {
		*s = Settings{}
		return nil
	}

	var data []byte
	switch v := val.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into Settings", val)
	}

	return json.Unmarshal(data, s)
}

func (s Settings) Value() (driver.Value, error) {
	return json.Marshal(s)
}
