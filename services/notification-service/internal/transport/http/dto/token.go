package dto

import "github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"

// CreateTokenRequest — регистрация устройства.
//
// deviceId клиент генерирует сам при первом запуске и хранит локально: он
// переживает смену FCM-токена, к нему и привязаны настройки уведомлений.
type CreateTokenRequest struct {
	Token    string `form:"token" json:"token" binding:"required,max=512"`
	DeviceID string `form:"deviceId" json:"deviceId" binding:"required,max=64"`
	Platform string `form:"platform" json:"platform" binding:"required,oneof=android ios web"`
}

func (r CreateTokenRequest) ToParams(userID uint) token.RegisterParams {
	return token.RegisterParams{
		UserID:   userID,
		DeviceID: r.DeviceID,
		Token:    r.Token,
		Platform: token.Platform(r.Platform),
	}
}

// UpdateSettingsRequest — частичное обновление настроек текущего устройства.
//
// Все поля опциональны: клиент присылает только то, что пользователь тронул.
// Отсутствие поля означает «не менять», а не «выключить».
type UpdateSettingsRequest struct {
	DeviceID string `form:"deviceId" json:"deviceId" binding:"required,max=64"`

	Enabled *bool `form:"enabled" json:"enabled"`

	// Muted — карта «тип уведомления → выключен». true выключает, false
	// включает обратно. Типы вне карты остаются как были.
	Muted map[string]bool `form:"muted" json:"muted"`
}

func (r UpdateSettingsRequest) ToParams(userID uint) token.UpdateSettingsParams {
	return token.UpdateSettingsParams{
		UserID:   userID,
		DeviceID: r.DeviceID,
		Enabled:  r.Enabled,
		Muted:    r.Muted,
	}
}

// SettingsResponse — настройки устройства после изменения.
type SettingsResponse struct {
	DeviceID string          `json:"deviceId"`
	Enabled  bool            `json:"enabled"`
	Muted    map[string]bool `json:"muted"`
}

func NewSettingsResponse(m *token.Model) SettingsResponse {
	muted := m.Settings.Muted
	if muted == nil {
		// Пустая карта, а не null: клиенту удобнее читать её без проверки на
		// отсутствие.
		muted = map[string]bool{}
	}

	return SettingsResponse{
		DeviceID: m.DeviceID,
		Enabled:  m.Settings.Enabled,
		Muted:    muted,
	}
}
