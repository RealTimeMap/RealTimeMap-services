package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"go.uber.org/zap"
)

type MessageSender interface {
	SendMessage(ctx context.Context, params services.MessageCreateParams) (*message.Message, error)
}

type MessageSenderHandler struct {
	sender        MessageSender
	profileGetter ProfileGetter
	logger        *zap.Logger
}

func NewMessageSenderHandler(
	sender MessageSender,
	profileGetter ProfileGetter,
	logger *zap.Logger) *MessageSenderHandler {
	return &MessageSenderHandler{
		sender:        sender,
		profileGetter: profileGetter,
		logger:        logger,
	}
}

type MessageCreateCommand struct {
	ChatID   uint
	SenderID uint

	Content string
}

func (h *MessageSenderHandler) Handle(ctx context.Context, cmd MessageCreateCommand) (MessageResult, error) {
	h.logger.Info("start chatUseCases.MessageSenderHandler.Handle",
		zap.Uint("chat_id", cmd.ChatID), zap.Uint("sender_id", cmd.SenderID))

	messObj, err := h.sender.SendMessage(ctx, services.MessageCreateParams{
		ChatID:   cmd.ChatID,
		SenderID: cmd.SenderID,
		Content:  cmd.Content,
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

	h.logger.Info("message sent", zap.Uint("message_id", messObj.ID), zap.Uint("chat_id", cmd.ChatID))
	return toMessageResult(messObj, prof), nil
}
