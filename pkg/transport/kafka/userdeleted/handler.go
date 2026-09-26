// Package userdeleted — обработчик user.deleted для сервисов, которым из
// топика auth-сервиса больше ничего не нужно.
//
// Сервис передаёт свой Eraser и получает готовый обработчик для
// consumer.New: разбор события, пропуск чужих типов и классификация ошибок
// одинаковы везде, отличается только то, что именно стирается.
package userdeleted

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Eraser стирает или обезличивает данные пользователя в хранилище сервиса.
//
// Обязан быть идемпотентным: at-least-once доставка привезёт событие
// повторно, а к тому моменту удалять уже нечего.
type Eraser interface {
	EraseUser(ctx context.Context, userID uint) error
}

// Handler обрабатывает user.deleted и молча пропускает остальные события
// топика: регистрации, входы, смену пароля.
//
// Сбой хранилища — Retryable: пропустить удаление значит оставить
// персональные данные, которые пользователь попросил стереть.
func Handler(eraser Eraser, logger *zap.Logger) consumer.MessageHandler {
	return func(ctx context.Context, msg kafka.Message) error {
		deleted, ok, err := events.ParseUserDeleted(msg.Value)
		if err != nil {
			logger.Warn("skipping malformed message", zap.Error(err))
			return consumer.Skip(err)
		}
		if !ok {
			return nil
		}

		if err := eraser.EraseUser(ctx, deleted.UserID); err != nil {
			logger.Error("failed to erase user data, will retry",
				zap.Uint("user_id", deleted.UserID), zap.Error(err))
			return consumer.Retryable(err)
		}

		logger.Info("user data erased", zap.Uint("user_id", deleted.UserID))
		return nil
	}
}
