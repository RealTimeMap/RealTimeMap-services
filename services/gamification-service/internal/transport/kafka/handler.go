// Package kafka принимает доменные события других сервисов, начисляет опыт и
// открывает достижения.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	pkgkafka "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/achievement"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/event"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Handler struct {
	service    *event.Service
	achService achievement.Service
	logger     *zap.Logger
}

func NewHandler(service *event.Service, achService achievement.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service:    service,
		achService: achService,
		logger:     logger,
	}
}

// eventMeta — то, что сервису нужно от события: кто и что сделал.
type eventMeta struct {
	EventType string
	UserID    uint
	SourceID  *uint
}

// HandleMessage начисляет опыт за событие и проверяет достижения.
func (h *Handler) HandleMessage(ctx context.Context, msg kafka.Message) error {
	meta, err := extractMeta(msg)
	if err != nil {
		h.logger.Warn("cannot extract event meta", zap.Error(err))
		return consumer.Skip(err)
	}

	log := h.logger.With(
		zap.String("event_type", meta.EventType),
		zap.Uint("user_id", meta.UserID),
	)
	log.Debug("received kafka message")

	if err := h.service.GreatUserExp(ctx, meta.UserID, meta.EventType, meta.SourceID); err != nil {
		// Правила нет или событие исчерпало дневной лимит — это штатный
		// исход, а не сбой: топик несёт события, за которые опыт не положен.
		// Достижения при этом всё равно проверяются: их пороги считаются по
		// счётчику событий, а не по начисленному опыту.
		if isExpected(err) {
			log.Debug("no xp credited for event", zap.Error(err))
		} else {
			log.Warn("failed to credit xp", zap.Error(err))
		}
	}

	h.achService.OnEvent(ctx, meta.UserID, meta.EventType)

	return nil
}

// isExpected отличает штатный отказ в начислении от настоящей ошибки.
//
// Отсутствие правила, выключенный конфиг и исчерпанный дневной лимит — все
// доменные ошибки со статусом ниже 500. Различие важно только для уровня
// лога: ни то ни другое не должно останавливать партицию, но отсутствие
// правила не должно выглядеть аварией — иначе реальные сбои утонут в шуме.
func isExpected(err error) bool {
	var domainErr apperror.DomainError
	return errors.As(err, &domainErr) && domainErr.HTTPStatus() < 500
}

// extractMeta достаёт мету события из тела, откатываясь на headers.
//
// Тело — основной источник: заголовки может потерять промежуточный компонент
// (mirror-maker, прокси). Headers остаются для сообщений, где тело не несёт
// нужных полей.
func extractMeta(msg kafka.Message) (eventMeta, error) {
	body := parseBody(msg.Value)
	headers := pkgkafka.ExtractMeta(msg)

	meta := eventMeta{
		EventType: firstNonEmpty(body.EventType, headers.EventType),
	}
	if meta.EventType == "" {
		return eventMeta{}, errors.New("event type is missing in both body and headers")
	}

	userID := body.UserID
	if userID == nil {
		parsed, err := parseUint(headers.UserID)
		if err != nil {
			return eventMeta{}, err
		}
		userID = parsed
	}
	if userID == nil {
		return eventMeta{}, errors.New("user id is missing in both body and headers")
	}
	meta.UserID = *userID

	sourceID := body.SourceID
	if sourceID == nil {
		parsed, err := parseUint(headers.SourceID)
		if err != nil {
			return eventMeta{}, err
		}
		sourceID = parsed
	}
	meta.SourceID = sourceID

	return meta, nil
}

// bodyMeta — поля, которые удаётся вытащить из тела события любого известного
// формата.
type bodyMeta struct {
	EventType string
	UserID    *uint
	SourceID  *uint
}

// parseBody разбирает тело, понимая три формы:
//
//   - конверт {type, payload:{userId, commentId}} — события Go-сервисов;
//   - конверт {type, payload:{user_id}} — события auth-сервиса;
//   - плоский {event_type, user_id} — формат auth-сервиса до перехода на
//     конверт, ещё лежащий в топике по retention.
//
// Разбор не возвращает ошибку: нечитаемое тело — не повод отбрасывать
// сообщение, пока мету можно взять из headers.
func parseBody(value []byte) bodyMeta {
	var raw struct {
		Type      string `json:"type"`
		EventType string `json:"event_type"`

		// Плоский формат кладёт идентификаторы в корень.
		UserID *uint `json:"user_id"`

		Payload struct {
			UserID      *uint `json:"userId"`
			UserIDSnake *uint `json:"user_id"`

			// Идентификатор сущности, которой касается событие. Имя поля
			// зависит от события, поэтому перечислены известные варианты.
			CommentID *uint `json:"commentId"`
			MarkID    *uint `json:"markId"`
			SourceID  *uint `json:"source_id"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(value, &raw); err != nil {
		return bodyMeta{}
	}

	return bodyMeta{
		EventType: firstNonEmpty(raw.Type, raw.EventType),
		UserID:    firstNonNil(raw.Payload.UserID, raw.Payload.UserIDSnake, raw.UserID),
		SourceID:  firstNonNil(raw.Payload.CommentID, raw.Payload.MarkID, raw.Payload.SourceID),
	}
}

func parseUint(s string) (*uint, error) {
	if s == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, err
	}
	id := uint(parsed)
	return &id, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonNil(values ...*uint) *uint {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}
