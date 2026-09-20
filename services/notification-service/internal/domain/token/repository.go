package token

import "context"

type Repository interface {
	// Register сохраняет устройство: создаёт новое либо обновляет токен уже
	// известного, не трогая его настройки.
	Register(ctx context.Context, obj *Model) error

	// GetByUser отдаёт все устройства пользователя, включая те, на которых
	// уведомления выключены: фильтрует домен, а не запрос.
	GetByUser(ctx context.Context, userID uint) ([]Model, error)

	// GetByDevice ищет устройство пользователя по клиентскому идентификатору.
	GetByDevice(ctx context.Context, userID uint, deviceID string) (*Model, error)

	// UpdateSettings сохраняет настройки устройства.
	UpdateSettings(ctx context.Context, id uint, settings Settings) error
}
