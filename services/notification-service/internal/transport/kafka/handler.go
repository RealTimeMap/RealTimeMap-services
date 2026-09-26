// Package kafka принимает доменные события других сервисов и превращает их в
// push-уведомления.
//
// Хендлер не решает, отправлять ли уведомление немедленно — это дело
// схлопывания (domain/collapse). Его задача: разобрать событие, определить
// адресата и собрать текст.
//
// Отправка идёт прямо в обработчике, а не через очередь в БД, как в
// smtp-service. Причина — FCM отвечает за десятки миллисекунд против секунд
// SMTP-диалога, а схлопывание уже срезает основной поток: серия из десяти
// сообщений даёт две отправки, а не десять. Когда объём вырастет, между
// хендлером и отправкой встанет очередь из docs/PLAN.md — интерфейс Notifier
// для этого уже отделён.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	pkgkafka "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/app/use_cases/notify"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/collapse"
)

// entityMark — значение CommentPayload.EntityType для комментариев к метке.
// Совпадает с comment.EntityMark на стороне comment-service; константа
// продублирована, чтобы не тянуть сюда его доменный пакет.
const entityMark = "mark_action"

// Notifier отправляет уведомление о событии. Интерфейс объявлен на стороне
// потребителя; *notify.EventNotifyHandler удовлетворяет ему напрямую.
type Notifier interface {
	Handle(ctx context.Context, cmd notify.NotifyEventCommand) error
}

// UserEraser удаляет устройства пользователя, удалившего аккаунт.
// *token.Service удовлетворяет ему напрямую.
type UserEraser interface {
	DeleteUser(ctx context.Context, userID uint) error
}

type Handler struct {
	notifier Notifier
	eraser   UserEraser
	logger   *zap.Logger
}

func NewHandler(notifier Notifier, eraser UserEraser, logger *zap.Logger) *Handler {
	return &Handler{notifier: notifier, eraser: eraser, logger: logger}
}

// HandleMessage разбирает сообщение и направляет его обработчику по типу.
func (h *Handler) HandleMessage(ctx context.Context, msg segmentio.Message) error {
	var raw events.RawEvent
	if err := json.Unmarshal(msg.Value, &raw); err != nil {
		// Нечитаемое тело — не повод перечитывать сообщение: повтор разберёт
		// его ровно так же. Коммитим и идём дальше.
		return consumer.Skip(fmt.Errorf("unmarshal event: %w", err))
	}

	// Тип берётся из тела, с откатом на headers: заголовки может потерять
	// промежуточный компонент (mirror-maker, прокси), тело — нет.
	eventType := raw.Type
	if eventType == "" {
		eventType = pkgkafka.ExtractMeta(msg).EventType
	}
	if eventType == "" {
		return consumer.Skip(errors.New("event type is missing in both body and headers"))
	}

	log := h.logger.With(
		zap.String("event_type", eventType),
		zap.String("topic", msg.Topic),
	)

	switch eventType {
	case events.UserDeleted:
		return h.handleUserDeleted(ctx, msg, log)
	case events.ChatMessageCreated:
		return h.handleChatMessage(ctx, raw.Payload, log)
	case events.CommentCreated:
		return h.handleComment(ctx, raw.Payload, log)
	case events.SubscriptionCreated:
		return h.handleSubscription(ctx, raw.Payload, log)
	default:
		// Топики общие: в них едут события, за которые уведомления не
		// положены. Это штатный исход, а не сбой.
		log.Debug("event type is not handled, skipping")
		return nil
	}
}

// handleUserDeleted удаляет push-токены удалённого аккаунта.
func (h *Handler) handleUserDeleted(ctx context.Context, msg segmentio.Message, log *zap.Logger) error {
	deleted, _, err := events.ParseUserDeleted(msg.Value)
	if err != nil {
		return consumer.Skip(err)
	}

	if err := h.eraser.DeleteUser(ctx, deleted.UserID); err != nil {
		log.Error("failed to delete device tokens, will retry",
			zap.Uint("user_id", deleted.UserID), zap.Error(err))
		return consumer.Retryable(err)
	}
	return nil
}

// handleChatMessage уведомляет участников чата о новом сообщении.
//
// Схлопывание идёт по чату: серия сообщений в одном чате — одна серия, а
// параллельная переписка в другом чате свой пуш получает независимо.
func (h *Handler) handleChatMessage(ctx context.Context, raw json.RawMessage, log *zap.Logger) error {
	var payload events.ChatMessagePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return consumer.Skip(fmt.Errorf("unmarshal chat message payload: %w", err))
	}

	if payload.ChatID == 0 || len(payload.RecipientIDs) == 0 {
		log.Debug("chat message has no recipients, skipping")
		return nil
	}

	title := payload.SenderName
	if payload.IsGroup && payload.ChatTitle != "" {
		// В группе заголовком служит имя чата, иначе по пушу не понять, куда
		// пришло сообщение: имён отправителей в группе много.
		title = payload.ChatTitle
	}
	if title == "" {
		title = "Новое сообщение"
	}

	content := payload.Preview
	if payload.IsGroup && payload.SenderName != "" && content != "" {
		content = payload.SenderName + ": " + content
	}
	if content == "" {
		content = "Новое сообщение"
	}

	var errs []error
	for _, recipientID := range payload.RecipientIDs {
		// Отправитель не уведомляется о собственном сообщении, даже если
		// продюсер по ошибке включил его в список.
		if recipientID == payload.SenderID {
			continue
		}

		err := h.notifier.Handle(ctx, notify.NotifyEventCommand{
			Key: collapse.Key{
				Kind:        collapse.KindChatMessage,
				RecipientID: recipientID,
				SourceID:    payload.ChatID,
			},
			Title:            title,
			Content:          content,
			CollapsedContent: "%d новых сообщений",
		})
		if err != nil {
			errs = append(errs, err)
		}
	}

	return h.resolve(errs, log, "chat message")
}

// handleComment уведомляет о новом комментарии к метке.
//
// Адресат — автор комментария, на который отвечают (ParentUserID). Владелец
// метки о комментариях верхнего уровня пока не уведомляется: mark-service не
// кладёт ownerId в событие комментария и не поднимает gRPC-ручку «метка по
// id» — спросить его не у кого. См. docs/PLAN.md.
func (h *Handler) handleComment(ctx context.Context, raw json.RawMessage, log *zap.Logger) error {
	var payload events.CommentPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return consumer.Skip(fmt.Errorf("unmarshal comment payload: %w", err))
	}

	if payload.EntityType != entityMark {
		log.Debug("comment is not attached to a mark, skipping",
			zap.String("entity_type", payload.EntityType))
		return nil
	}

	if payload.ParentUserID == nil {
		log.Debug("top-level comment, no recipient to notify",
			zap.Uint("comment_id", payload.CommentID))
		return nil
	}

	recipientID := *payload.ParentUserID
	// Ответ самому себе уведомления не порождает.
	if recipientID == payload.UserID {
		return nil
	}

	title := payload.Username
	if title == "" {
		title = "Новый комментарий"
	}

	content := preview(payload.Content, 120)
	if content == "" {
		content = "ответил на ваш комментарий"
	}

	err := h.notifier.Handle(ctx, notify.NotifyEventCommand{
		Key: collapse.Key{
			Kind:        collapse.KindComment,
			RecipientID: recipientID,
			// Серия считается по метке, а не по комментарию: десять ответов
			// под одной меткой — одна серия. Ключ по CommentID схлопывал бы
			// только повторы одного и того же комментария, то есть ничего.
			SourceID: payload.EntityID,
		},
		Title:            title,
		Content:          content,
		CollapsedContent: "%d новых комментариев",
	})

	return h.resolve(errorsOf(err), log, "comment")
}

// handleSubscription уведомляет о новом подписчике.
func (h *Handler) handleSubscription(ctx context.Context, raw json.RawMessage, log *zap.Logger) error {
	var payload events.SubscriptionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return consumer.Skip(fmt.Errorf("unmarshal subscription payload: %w", err))
	}

	if payload.TargetID == 0 || payload.TargetID == payload.SubscriberID {
		return nil
	}

	content := "У вас новый подписчик"
	if payload.SubscriberName != "" {
		content = payload.SubscriberName + " подписался на вас"
	}

	err := h.notifier.Handle(ctx, notify.NotifyEventCommand{
		Key: collapse.Key{
			Kind:        collapse.KindSubscriber,
			RecipientID: payload.TargetID,
			// Источника нет: подписчики схлопываются в одну серию на
			// получателя независимо от того, кто именно подписался.
			SourceID: 0,
		},
		Title:            "Новый подписчик",
		Content:          content,
		CollapsedContent: "%d новых подписчиков",
	})

	return h.resolve(errorsOf(err), log, "subscription")
}

// resolve превращает ошибки отправки в вердикт для consumer.
//
// Отсутствие токенов — не ошибка доставки: у пользователя просто нет
// зарегистрированного устройства, и повтор сообщения ничего не изменит.
// Остальные сбои отдаются как retryable: уведомление о событии имеет смысл
// только вовремя, но потерять его из-за моргнувшей сети хуже, чем прислать с
// задержкой в секунду.
func (h *Handler) resolve(errs []error, log *zap.Logger, kind string) error {
	var real []error
	for _, err := range errs {
		if err == nil || isNoTokens(err) {
			continue
		}
		real = append(real, err)
	}

	if len(real) == 0 {
		return nil
	}

	log.Warn("failed to deliver notifications",
		zap.String("kind", kind),
		zap.Int("failed", len(real)),
		zap.Error(real[0]),
	)
	return consumer.Retryable(errors.Join(real...))
}

// isNoTokens отличает «у пользователя нет устройств» от сбоя доставки.
func isNoTokens(err error) bool {
	var notFound *apperror.NotFoundError
	return errors.As(err, &notFound)
}

func errorsOf(err error) []error {
	if err == nil {
		return nil
	}
	return []error{err}
}

// preview усекает текст до лимита по рунам, не разрезая слово пополам.
func preview(s string, limit int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}

	cut := string(runes[:limit])
	if idx := strings.LastIndex(cut, " "); idx > limit/2 {
		cut = cut[:idx]
	}
	return strings.TrimSpace(cut) + "…"
}
