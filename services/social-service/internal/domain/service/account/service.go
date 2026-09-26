// Package account стирает данные пользователя, удалившего аккаунт.
package account

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Eraser удаляет данные пользователя из хранилища одной транзакцией.
//
// Возвращает пользователей, чьи счётчики (друзья, подписки, подписчики)
// изменились, — их кеш статистики нужно сбросить.
//
// Что происходит с данными:
//   - профиль, дружбы, подписки и блокировки в обе стороны — удаляются;
//   - участие в чатах — удаляется;
//   - сообщения остаются у собеседников, но обезличиваются: sender_id = 0.
//
// Повторный вызов для того же пользователя безопасен и ничего не меняет.
type Eraser interface {
	EraseUser(ctx context.Context, userID uint) (affected []uint, err error)
}

// StatInvalidator сбрасывает кеш статистики профилей.
//
// Порт объявлен у потребителя: доменный слой не знает про Redis.
type StatInvalidator interface {
	InvalidateProfiles(ctx context.Context, profileIDs ...uint)
}

type noOpStatInvalidator struct{}

func (noOpStatInvalidator) InvalidateProfiles(context.Context, ...uint) {}

type Service struct {
	eraser    Eraser
	statCache StatInvalidator

	logger *zap.Logger
}

func NewService(eraser Eraser, statCache StatInvalidator, logger *zap.Logger) *Service {
	if statCache == nil {
		statCache = noOpStatInvalidator{}
	}
	return &Service{eraser: eraser, statCache: statCache, logger: logger}
}

// DeleteUser стирает данные пользователя в social-service.
func (s *Service) DeleteUser(ctx context.Context, userID uint) error {
	affected, err := s.eraser.EraseUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("erase user %d: %w", userID, err)
	}

	// Кеш сбрасывается после коммита: сброс до него позволил бы
	// параллельному запросу закешировать старые счётчики заново.
	s.statCache.InvalidateProfiles(ctx, append(affected, userID)...)

	s.logger.Info("user data erased",
		zap.Uint("user_id", userID),
		zap.Int("affected_profiles", len(affected)))

	return nil
}
