package chat

import (
	"context"

	"go.uber.org/zap"
)

// ChatReader — порт доменного сервиса для отметки чата прочитанным. Возвращает
// id сообщения, на которое встал курсор, и moved — было ли реальное движение
// (false, если в чате нет сообщений).
type ChatReader interface {
	MarkRead(ctx context.Context, chatID, userID uint) (lastReadMessageID uint, moved bool, err error)
}

type MarkReadHandler struct {
	reader ChatReader
	events EventPublisher
	logger *zap.Logger
}

func NewMarkReadHandler(reader ChatReader, events EventPublisher, logger *zap.Logger) *MarkReadHandler {
	return &MarkReadHandler{
		reader: reader,
		events: events,
		logger: logger,
	}
}

type MarkReadCommand struct {
	ChatID uint
	UserID uint
}

// Handle двигает курсор прочтения пользователя на последнее сообщение чата.
// Идемпотентно: повторный вызов ничего не ломает (курсор монотонный). После
// реального продвижения курсора публикует chat.read в комнату чата — собеседник
// видит галочки, другие устройства читающего обнуляют unread.
func (h *MarkReadHandler) Handle(ctx context.Context, cmd MarkReadCommand) error {
	h.logger.Info("start chatUseCases.MarkReadHandler.Handle",
		zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))

	lastReadID, moved, err := h.reader.MarkRead(ctx, cmd.ChatID, cmd.UserID)
	if err != nil {
		h.logger.Warn("failed to mark chat read", zap.Error(err),
			zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))
		return err
	}

	// В чате нет сообщений — курсор не двигали, оповещать нечем.
	if !moved {
		return nil
	}

	h.publishRead(ctx, cmd.ChatID, cmd.UserID, lastReadID)

	h.logger.Info("chat marked read", zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))
	return nil
}

// publishRead рассылает chat.read в комнату чата (все участники + другие
// устройства читающего). Best-effort: курсор уже сохранён, ошибка публикации
// только логируется и не влияет на HTTP-ответ.
func (h *MarkReadHandler) publishRead(ctx context.Context, chatID, userID, lastReadID uint) {
	if err := h.events.Publish(ctx, ChatEvent{
		Type:   EventChatRead,
		ChatID: chatID,
		Payload: ReadResult{
			ChatID:            chatID,
			UserID:            userID,
			LastReadMessageID: lastReadID,
		},
	}); err != nil {
		h.logger.Warn("failed to publish chat.read event",
			zap.Error(err), zap.Uint("chat_id", chatID), zap.Uint("user_id", userID))
	}
}
