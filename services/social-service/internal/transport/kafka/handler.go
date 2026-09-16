// Package kafka принимает доменные события auth-сервиса и заводит профиль
// пользователя.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	pkgkafka "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Handler struct {
	service *profile.Service

	logger *zap.Logger
}

func NewHandler(service *profile.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// HandleMessage разбирает сообщение и заводит профиль на регистрацию.
//
// Топик auth-сервиса общий: туда же едут вход в аккаунт, смена пароля и
// подтверждение адреса. Профиль заводится только на user.registered —
// без проверки типа любое из этих событий создавало бы профиль повторно.
func (h *Handler) HandleMessage(ctx context.Context, msg kafka.Message) error {
	eventType, payload, err := decodeEvent(msg)
	if err != nil {
		h.logger.Error("Error unmarshalling message", zap.Error(err))
		return consumer.Skip(err)
	}

	if eventType != events.UserRegistered {
		// Событие чужого типа — не наше дело.
		return nil
	}

	user, err := decodeRegistered(payload)
	if err != nil {
		h.logger.Warn("skipping user.registered: malformed payload", zap.Error(err))
		return consumer.Skip(err)
	}

	h.logger.Info("Message received",
		zap.Uint("user_id", user.UserID),
		zap.String("username", user.Username))

	_, err = h.service.CreateProfile(ctx, profile.CreateProfileInput{
		Username: user.Username,
		UserID:   user.UserID,
	})
	if err != nil {
		var conflictErr *apperror.ConflictError
		if errors.As(err, &conflictErr) {
			h.logger.Info("profile already exists, skipping",
				zap.Uint("user_id", user.UserID))
			return consumer.Skip(err)
		}

		h.logger.Error("Error creating profile", zap.Error(err))
		return consumer.Retryable(err)
	}

	return nil
}

// registeredUser — то, что сервису нужно от события регистрации.
type registeredUser struct {
	UserID   uint
	Username string
}

// decodeEvent достаёт тип события и неразобранный payload.
//
// Тип берётся из тела, а заголовок остаётся запасным вариантом: headers может
// потерять промежуточный компонент (mirror-maker, прокси), тело — нет.
// Читаются оба ключа: "type" — общий конверт, "event_type" — плоский формат,
// в котором auth-сервис слал регистрацию до перехода на конверт.
func decodeEvent(msg kafka.Message) (string, json.RawMessage, error) {
	var envelope struct {
		Type      string          `json:"type"`
		EventType string          `json:"event_type"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return "", nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	eventType := envelope.Type
	if eventType == "" {
		eventType = envelope.EventType
	}
	if eventType == "" {
		eventType = pkgkafka.ExtractMeta(msg).EventType
	}

	// Плоский формат кладёт поля в корень: payload там нет, разбирать надо
	// само тело.
	payload := envelope.Payload
	if len(payload) == 0 {
		payload = msg.Value
	}

	return eventType, payload, nil
}

// decodeRegistered разбирает payload регистрации.
//
// user_id обязателен: профиль с нулевым идентификатором не принадлежит
// никому, а повтор такого сообщения упрётся в конфликт по первому заведённому.
func decodeRegistered(payload json.RawMessage) (registeredUser, error) {
	var raw struct {
		UserID   *uint  `json:"user_id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return registeredUser{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	if raw.UserID == nil || *raw.UserID == 0 {
		return registeredUser{}, errors.New("user id is missing in payload")
	}

	return registeredUser{
		UserID:   *raw.UserID,
		Username: raw.Username,
	}, nil
}
