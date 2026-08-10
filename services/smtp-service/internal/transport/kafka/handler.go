// Package kafka принимает доменные события других сервисов и ставит письма
// в очередь.
//
// Хендлер не отправляет писем сам: он делает INSERT и отдаёт управление, после
// чего consumer коммитит offset. Отправка внутри обработчика заблокировала бы
// партицию на время SMTP-диалога — сотни миллисекунд в норме и секунды
// таймаута при недоступности провайдера.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	pkgkafka "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Enqueuer ставит письмо в очередь. Интерфейс объявлен на стороне
// потребителя; *email.Service удовлетворяет ему напрямую.
type Enqueuer interface {
	Enqueue(ctx context.Context, in email.EnqueueInput) (*email.EnqueueResult, error)
}

type Handler struct {
	emails Enqueuer
	logger *zap.Logger
}

func NewHandler(emails Enqueuer, logger *zap.Logger) *Handler {
	return &Handler{
		emails: emails,
		logger: logger,
	}
}

// HandleMessage разбирает сообщение и направляет его обработчику по типу.
func (h *Handler) HandleMessage(ctx context.Context, msg segmentio.Message) error {
	// Тип события берётся из тела; заголовок используется как запасной
	// вариант — часть продюсеров в проекте кладёт его только туда.
	var envelope struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return consumer.Skip(fmt.Errorf("unmarshal envelope: %w", err))
	}

	eventType := envelope.EventType
	if eventType == "" {
		eventType = pkgkafka.ExtractMeta(msg).EventType
	}

	switch eventType {
	case EventUserRegistered:
		return h.handleUserRegistered(ctx, msg)
	default:
		// Топик может содержать события, до которых сервису нет дела.
		return nil
	}
}

func (h *Handler) handleUserRegistered(ctx context.Context, msg segmentio.Message) error {
	var event UserRegistered
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return consumer.Skip(fmt.Errorf("unmarshal user.registered: %w", err))
	}

	traceID := pkgkafka.GetHeader(msg, "X-Trace-Id")
	if traceID == "" {
		traceID = pkgkafka.GetHeader(msg, "trace_id")
	}

	res, err := h.emails.Enqueue(ctx, email.EnqueueInput{
		TemplateID: "welcome",
		ToEmail:    event.Email,
		UserID:     &event.UserID,
		Data: map[string]any{
			"username": event.Username,
		},
		// Явный ключ вместо хеша содержимого: смысл «одно приветственное
		// письмо на пользователя» не должен зависеть от набора полей события.
		// Хеш включил бы registered_at и рассыпался бы, начни продюсер слать
		// его иначе.
		IdempotencyKey: fmt.Sprintf("%s:%d", EventUserRegistered, event.UserID),
		TraceID:        traceID,
	})
	if err != nil {
		return h.classifyEnqueueError(err, event)
	}

	if res.Duplicate {
		h.logger.Debug("welcome email already queued",
			zap.Uint64("user_id", event.UserID),
			zap.String("email_id", res.EmailID.String()),
		)
		return nil
	}

	h.logger.Info("welcome email queued",
		zap.Uint64("user_id", event.UserID),
		zap.String("email_id", res.EmailID.String()),
		zap.String("to", email.MaskEmail(event.Email)),
	)

	return nil
}

// classifyEnqueueError решает, коммитить offset или перечитать сообщение.
//
// Разница существенная: Skip коммитит и идёт дальше, Retryable оставляет
// offset на месте и останавливает партицию до восстановления. Ошибку данных
// нельзя объявлять retryable — партиция встанет навсегда.
func (h *Handler) classifyEnqueueError(err error, event UserRegistered) error {
	log := h.logger.With(
		zap.Uint64("user_id", event.UserID),
		zap.String("to", email.MaskEmail(event.Email)),
		zap.Error(err),
	)

	// Битый адрес, отсутствующий шаблон, нехватка данных — повтором не
	// лечатся: сообщение в топике не изменится.
	var domainErr apperror.DomainError
	if errors.As(err, &domainErr) && domainErr.HTTPStatus() < 500 {
		log.Warn("skipping user.registered: email cannot be built from this event")
		return consumer.Skip(err)
	}

	// Остальное — недоступность БД: сообщение перечитается.
	//
	// Retryable останавливает партицию до восстановления, поэтому им нельзя
	// помечать ошибки данных: топик встал бы навсегда на одном письме.
	log.Error("failed to queue welcome email, will retry")
	return consumer.Retryable(err)
}
