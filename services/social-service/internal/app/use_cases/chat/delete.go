package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"go.uber.org/zap"
)

// ChatDeleter — порт доменного сервиса для удаления чата.
type ChatDeleter interface {
	Delete(ctx context.Context, chatID, userID uint, forEveryone bool) (*services.DeleteResult, error)
}

type DeleteHandler struct {
	deleter ChatDeleter
	rooms   EventPublisher
	logger  *zap.Logger
}

func NewDeleteHandler(deleter ChatDeleter, rooms EventPublisher, logger *zap.Logger) *DeleteHandler {
	return &DeleteHandler{
		deleter: deleter,
		rooms:   rooms,
		logger:  logger,
	}
}

type DeleteCommand struct {
	ChatID uint
	UserID uint

	// ForEveryone — удалить direct-чат у обоих собеседников. Для группы
	// игнорируется: её владелец удаляет её всегда у всех.
	ForEveryone bool
}

// Handle удаляет чат и синхронизирует клиентов.
//
// chat.deleted рассылается адресно, в комнаты user:<id>: после удаления у всех
// комната чата уже бесполезна, а при удалении «у себя» событие нужно только
// другим устройствам самого удалившего — собеседник о нём не узнаёт.
//
// При удалении у всех сокеты участников выводятся из комнаты chat:<id>.
// Realtime-часть best-effort: чат уже удалён в БД, и клиент, не получивший
// события, увидит это при следующей загрузке списка.
func (h *DeleteHandler) Handle(ctx context.Context, cmd DeleteCommand) error {
	res, err := h.deleter.Delete(ctx, cmd.ChatID, cmd.UserID, cmd.ForEveryone)
	if err != nil {
		h.logger.Warn("failed to delete chat", zap.Error(err),
			zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))
		return err
	}

	if err := h.rooms.Publish(ctx, ChatEvent{
		Type:         EventChatDeleted,
		RecipientIDs: res.ParticipantIDs,
		Payload: DeletedResult{
			ChatID:      cmd.ChatID,
			DeletedBy:   cmd.UserID,
			ForEveryone: res.ForEveryone,
		},
	}); err != nil {
		h.logger.Warn("failed to publish chat.deleted", zap.Error(err), zap.Uint("chat_id", cmd.ChatID))
	}

	if res.ForEveryone {
		if err := h.rooms.LeaveUsers(ctx, cmd.ChatID, res.ParticipantIDs); err != nil {
			h.logger.Warn("failed to sync rooms on delete", zap.Error(err), zap.Uint("chat_id", cmd.ChatID))
		}
	}

	h.logger.Info("chat deleted",
		zap.Uint("chat_id", cmd.ChatID),
		zap.Uint("user_id", cmd.UserID),
		zap.Bool("for_everyone", res.ForEveryone))
	return nil
}
