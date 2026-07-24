package chat

import (
	"context"

	"go.uber.org/zap"
)

// ChatLeaver — порт доменного сервиса для выхода пользователя из чата.
type ChatLeaver interface {
	Leave(ctx context.Context, chatID, userID uint) error
}

type LeaveHandler struct {
	leaver ChatLeaver
	rooms  EventPublisher
	logger *zap.Logger
}

func NewLeaveHandler(leaver ChatLeaver, rooms EventPublisher, logger *zap.Logger) *LeaveHandler {
	return &LeaveHandler{
		leaver: leaver,
		rooms:  rooms,
		logger: logger,
	}
}

type LeaveCommand struct {
	ChatID uint
	UserID uint
}

// Handle выводит пользователя из чата и синхронизирует комнаты: сокеты вышедшего
// немедленно покидают chat:<id> и перестают получать события чата (дыра
// приватности закрыта). Room-sync — best-effort: даже если он не сработал,
// участник уже помечен вышедшим в БД и новых сообщений в истории не увидит.
func (h *LeaveHandler) Handle(ctx context.Context, cmd LeaveCommand) error {
	h.logger.Info("start chatUseCases.LeaveHandler.Handle",
		zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))

	if err := h.leaver.Leave(ctx, cmd.ChatID, cmd.UserID); err != nil {
		h.logger.Warn("failed to leave chat", zap.Error(err),
			zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))
		return err
	}

	if err := h.rooms.LeaveUsers(ctx, cmd.ChatID, []uint{cmd.UserID}); err != nil {
		h.logger.Warn("failed to sync rooms on leave", zap.Error(err),
			zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))
	}

	h.logger.Info("user left chat", zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))
	return nil
}
