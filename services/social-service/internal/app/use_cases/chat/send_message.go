package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"go.uber.org/zap"
)

type MessageSender interface {
	SendMessage(ctx context.Context, params services.MessageCreateParams) (*message.Message, error)
	// RecipientIDs — все активные участники чата (включая отправителя), кому
	// доставить realtime-событие. Клиент-инициатор дедупит эхо по id.
	RecipientIDs(ctx context.Context, chatID uint) ([]uint, error)
}

type MessageSenderHandler struct {
	sender        MessageSender
	profileGetter ProfileGetter
	events        EventPublisher
	logger        *zap.Logger
}

func NewMessageSenderHandler(
	sender MessageSender,
	profileGetter ProfileGetter,
	events EventPublisher,
	logger *zap.Logger) *MessageSenderHandler {
	return &MessageSenderHandler{
		sender:        sender,
		profileGetter: profileGetter,
		events:        events,
		logger:        logger,
	}
}

type MessageCreateCommand struct {
	ChatID   uint
	SenderID uint

	Content string

	// ClientMessageID — идемпотентный ключ от клиента (UUID). Опционален.
	ClientMessageID string
}

func (h *MessageSenderHandler) Handle(ctx context.Context, cmd MessageCreateCommand) (MessageResult, error) {
	h.logger.Info("start chatUseCases.MessageSenderHandler.Handle",
		zap.Uint("chat_id", cmd.ChatID), zap.Uint("sender_id", cmd.SenderID))

	messObj, err := h.sender.SendMessage(ctx, services.MessageCreateParams{
		ChatID:          cmd.ChatID,
		SenderID:        cmd.SenderID,
		Content:         cmd.Content,
		ClientMessageID: cmd.ClientMessageID,
	})
	if err != nil {
		h.logger.Warn("failed to send message", zap.Error(err),
			zap.Uint("chat_id", cmd.ChatID), zap.Uint("sender_id", cmd.SenderID))
		return MessageResult{}, err
	}

	// Сообщение сохранено. Обогащение профилем
	prof, err := h.profileGetter.GetProfile(ctx, cmd.SenderID)
	if err != nil {
		h.logger.Warn("failed to enrich message with sender profile",
			zap.Error(err), zap.Uint("sender_id", cmd.SenderID))
	}

	result := toMessageResult(messObj, prof)

	// Realtime-доставка всем участникам чата (включая отправителя — на его другие
	// устройства). Best-effort: сообщение уже сохранено, поэтому ошибка публикации
	// только логируется и не влияет на ответ (иначе клиент ретраит → дубликат).
	h.publishNewMessage(ctx, cmd.ChatID, result)

	h.logger.Info("message sent", zap.Uint("message_id", messObj.ID), zap.Uint("chat_id", cmd.ChatID))
	return result, nil
}

// publishNewMessage рассылает событие message.new всем участникам чата (включая
// отправителя — эхо на его другие устройства). Никогда не возвращает ошибку
// наверх — realtime не критичен для успеха HTTP-запроса.
func (h *MessageSenderHandler) publishNewMessage(ctx context.Context, chatID uint, payload MessageResult) {
	recipientIDs, err := h.sender.RecipientIDs(ctx, chatID)
	if err != nil {
		h.logger.Warn("failed to resolve recipients for realtime event",
			zap.Error(err), zap.Uint("chat_id", chatID))
		return
	}
	if len(recipientIDs) == 0 {
		return
	}

	if err := h.events.Publish(ctx, ChatEvent{
		Type:         EventMessageNew,
		RecipientIDs: recipientIDs,
		Payload:      payload,
	}); err != nil {
		h.logger.Warn("failed to publish message.new event",
			zap.Error(err), zap.Uint("chat_id", chatID))
	}
}
