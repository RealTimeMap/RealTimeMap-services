// Package kafka принимает доменные события auth-сервиса: заводит профиль
// пользователя и поддерживает признак администратора в актуальном состоянии.
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

// HandleMessage разбирает сообщение и разводит его по типу события.
//
// Топик auth-сервиса общий: туда же едут вход в аккаунт, смена пароля и
// подтверждение адреса. Обрабатываются только знакомые типы — без проверки
// любое из этих событий заводило бы профиль повторно.
func (h *Handler) HandleMessage(ctx context.Context, msg kafka.Message) error {
	eventType, payload, err := decodeEvent(msg)
	if err != nil {
		h.logger.Error("Error unmarshalling message", zap.Error(err))
		return consumer.Skip(err)
	}

	switch eventType {
	case events.UserRegistered:
		return h.handleRegistered(ctx, payload)
	case events.UserUpdated:
		return h.handleUpdated(ctx, payload)
	default:
		// Событие чужого типа — не наше дело.
		return nil
	}
}

// handleRegistered заводит профиль на регистрацию пользователя.
func (h *Handler) handleRegistered(ctx context.Context, payload json.RawMessage) error {
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
		IsAdmin:  user.IsAdmin,
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

// handleUpdated применяет признак администратора, изменённый в auth-сервисе.
//
// Остальные поля user.updated игнорируются: username, tag и аватар
// редактируются здесь, и запись их значений из auth затёрла бы свежую правку
// профиля тем, что auth знал на момент регистрации.
func (h *Handler) handleUpdated(ctx context.Context, payload json.RawMessage) error {
	user, err := decodeUpdated(payload)
	if err != nil {
		h.logger.Warn("skipping user.updated: malformed payload", zap.Error(err))
		return consumer.Skip(err)
	}

	// Событие без is_admin не о правах — применять нечего.
	if user.IsAdmin == nil {
		return nil
	}

	h.logger.Info("admin flag update received",
		zap.Uint("user_id", user.UserID),
		zap.Bool("is_admin", *user.IsAdmin))

	_, err = h.service.SyncAdmin(ctx, profile.SyncAdminInput{
		UserID:  user.UserID,
		IsAdmin: *user.IsAdmin,
	})
	if err != nil {
		var notFoundErr *apperror.NotFoundError
		if errors.As(err, &notFoundErr) {
			// Профиля ещё нет: user.updated обогнал user.registered либо
			// пользователь заведён в обход регистрации. Повтор не поможет —
			// профиль появится своим событием, уже с актуальным признаком.
			h.logger.Warn("profile not found, skipping admin sync",
				zap.Uint("user_id", user.UserID))
			return consumer.Skip(err)
		}

		h.logger.Error("Error syncing admin flag", zap.Error(err))
		return consumer.Retryable(err)
	}

	return nil
}

// registeredUser — то, что сервису нужно от события регистрации.
type registeredUser struct {
	UserID   uint
	Username string
	IsAdmin  bool
}

// updatedUser — то, что сервису нужно от события изменения пользователя.
//
// IsAdmin — указатель: отсутствующий ключ (событие про что-то другое) нужно
// отличать от явного false, которым снимают админку.
type updatedUser struct {
	UserID  uint
	IsAdmin *bool
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
		IsAdmin  bool   `json:"is_admin"`
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
		IsAdmin:  raw.IsAdmin,
	}, nil
}

// decodeUpdated разбирает payload изменения пользователя.
func decodeUpdated(payload json.RawMessage) (updatedUser, error) {
	var raw struct {
		UserID  *uint `json:"user_id"`
		IsAdmin *bool `json:"is_admin"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return updatedUser{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	if raw.UserID == nil || *raw.UserID == 0 {
		return updatedUser{}, errors.New("user id is missing in payload")
	}

	return updatedUser{
		UserID:  *raw.UserID,
		IsAdmin: raw.IsAdmin,
	}, nil
}
