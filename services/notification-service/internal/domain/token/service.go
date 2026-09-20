package token

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Service struct {
	repo Repository

	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

type RegisterParams struct {
	UserID   uint
	DeviceID string
	Token    string
	Platform Platform
}

// Register регистрирует устройство пользователя.
//
// Вызывается при каждом запуске приложения, а не только при первом: клиент не
// знает, сменился ли токен с прошлого раза, и переотправляет его всегда.
// Поэтому идемпотентность — требование, а не приятное свойство.
func (s *Service) Register(ctx context.Context, params RegisterParams) error {
	s.logger.Info("start Register",
		zap.String("layer", "token service"),
		zap.Uint("userID", params.UserID),
		zap.String("deviceID", params.DeviceID),
	)

	if !params.Platform.Valid() {
		return ErrInvalidPlatform(string(params.Platform))
	}

	now := time.Now()
	payload := &Model{
		UserID:     params.UserID,
		DeviceID:   params.DeviceID,
		Token:      params.Token,
		Platform:   params.Platform,
		LastUsedAt: &now,

		// Дефолт применится только при вставке: в списке обновляемых колонок
		// репозитория settings нет.
		Settings: DefaultSettings(),
	}

	return s.repo.Register(ctx, payload)
}

// GetUserDevices отдаёт устройства пользователя.
//
// Отсутствие устройств — доменная ошибка, а не пустой список: для отправки это
// означает «доставлять некуда», и вызывающий код отличает этот случай от сбоя
// по типу ошибки.
func (s *Service) GetUserDevices(ctx context.Context, userID uint) ([]Model, error) {
	s.logger.Info("start GetUserDevices", zap.String("layer", "token service"), zap.Uint("userID", userID))

	objs, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(objs) < 1 {
		return nil, ErrNotFoundUserToken(userID)
	}
	return objs, nil
}

// UpdateSettingsParams — частичное обновление настроек устройства.
//
// Указатели и карта изменений вместо готового Settings: клиент присылает
// только то, что тронул пользователь. Прислать настройки целиком он не может
// без гонки — между чтением и записью их мог изменить другой клиент того же
// устройства (приложение и его расширение, две вкладки).
type UpdateSettingsParams struct {
	UserID   uint
	DeviceID string

	Enabled *bool

	// Muted — изменения по типам: true выключает тип, false включает обратно.
	// Типы, которых в карте нет, остаются как были.
	Muted map[string]bool
}

func (s *Service) UpdateSettings(ctx context.Context, params UpdateSettingsParams) (*Model, error) {
	s.logger.Info("start UpdateSettings",
		zap.String("layer", "token service"),
		zap.Uint("userID", params.UserID),
		zap.String("deviceID", params.DeviceID),
	)

	for k := range params.Muted {
		if !Kind(k).Valid() {
			return nil, ErrUnknownKind(k)
		}
	}

	device, err := s.repo.GetByDevice(ctx, params.UserID, params.DeviceID)
	if err != nil {
		return nil, err
	}

	settings := device.Settings
	if params.Enabled != nil {
		settings.Enabled = *params.Enabled
	}

	for kind, muted := range params.Muted {
		if !muted {
			// Включённый тип не хранится: Muted перечисляет только
			// выключенное, и запись kind=false раздувала бы её до списка
			// всех существующих типов.
			delete(settings.Muted, kind)
			continue
		}
		if settings.Muted == nil {
			settings.Muted = make(map[string]bool, len(params.Muted))
		}
		settings.Muted[kind] = true
	}

	if err := s.repo.UpdateSettings(ctx, device.ID, settings); err != nil {
		return nil, err
	}

	device.Settings = settings
	return device, nil
}
