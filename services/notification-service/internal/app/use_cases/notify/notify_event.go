package notify

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/collapse"
)

// Limiter схлопывает серии однотипных уведомлений. Интерфейс объявлен на
// стороне потребителя; реализация из infrastructure/collapse удовлетворяет
// ему напрямую.
type Limiter interface {
	Observe(ctx context.Context, key collapse.Key, window time.Duration) (collapse.Decision, error)
}

// EventNotifyHandler отправляет уведомление о доменном событии, пропуская его
// через схлопывание.
//
// Отдельный обработчик поверх UserNotifyHanlder, а не логика внутри него:
// прямая отправка (тест, административная рассылка) схлопываться не должна —
// она не серия.
type EventNotifyHandler struct {
	notifier *UserNotifyHanlder
	limiter  Limiter

	// window — окно тишины. Одно на все типы: разводить его по Kind стоит
	// тогда, когда появится продуктовое основание, а не заранее.
	window time.Duration

	logger *zap.Logger
}

func NewEventNotifyHandler(
	notifier *UserNotifyHanlder,
	limiter Limiter,
	window time.Duration,
	logger *zap.Logger,
) *EventNotifyHandler {
	return &EventNotifyHandler{
		notifier: notifier,
		limiter:  limiter,
		window:   window,
		logger:   logger,
	}
}

// NotifyEventCommand — уведомление о событии, подлежащее схлопыванию.
type NotifyEventCommand struct {
	Key collapse.Key

	Title   string
	Content string

	// CollapsedContent — текст сводки. %d подставляется числом событий в
	// серии. Пустая строка отключает сводку для этого типа: подписчиков
	// удобнее показать одним «N новых подписчиков», а вот одиночные события
	// без осмысленного множественного текста лучше просто не досылать.
	CollapsedContent string
}

// Handle решает, отправлять ли уведомление, и отправляет.
//
// Сбой схлопывания не отменяет уведомление: Redis недоступен — шлём как есть.
// Лишний пуш безобиднее пропавшего.
func (h *EventNotifyHandler) Handle(ctx context.Context, cmd NotifyEventCommand) error {
	log := h.logger.With(
		zap.String("collapse_key", cmd.Key.String()),
		zap.Uint("user_id", cmd.Key.RecipientID),
	)

	decision, err := h.limiter.Observe(ctx, cmd.Key, h.window)
	if err != nil {
		log.Warn("collapse check failed, sending anyway", zap.Error(err))
		decision = collapse.Decision{Send: true}
	}

	if !decision.Send {
		log.Debug("notification collapsed")
		return nil
	}

	return h.notifier.Handle(ctx, NotifyUserCommand{
		UserID:  cmd.Key.RecipientID,
		Title:   cmd.Title,
		Content: cmd.Content,
	})
}

// HandleSummary отправляет сводку по закрытой серии: «N новых сообщений».
func (h *EventNotifyHandler) HandleSummary(ctx context.Context, key collapse.Key, title, template string, count int) error {
	if count < 1 || template == "" {
		return nil
	}

	return h.notifier.Handle(ctx, NotifyUserCommand{
		UserID:  key.RecipientID,
		Title:   title,
		Content: fmt.Sprintf(template, count),
	})
}
