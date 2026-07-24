package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"go.uber.org/zap"
)

type DirectManager interface {
	OpenDirectChat(ctx context.Context, params services.ChatCreateParams) (*chat.Chat, error)
}

type DirectChatHandler struct {
	manager    DirectManager
	profGetter ProfileGetter
	rooms      EventPublisher
	logger     *zap.Logger
}

func NewDirectChatHandler(manager DirectManager, profGetter ProfileGetter, rooms EventPublisher, logger *zap.Logger) *DirectChatHandler {
	return &DirectChatHandler{
		manager:    manager,
		profGetter: profGetter,
		rooms:      rooms,
		logger:     logger,
	}
}

type CreateDirectCommand struct {
	UserID uint
	PeerID uint
}

func (h *DirectChatHandler) Handle(ctx context.Context, cmd CreateDirectCommand) (DirectChatResult, error) {
	h.logger.Info("start chatUseCases.DirectChatHandler.Handle",
		zap.Uint("user_id", cmd.UserID), zap.Uint("peer_id", cmd.PeerID))

	prof, err := h.profGetter.GetProfile(ctx, cmd.PeerID)
	if err != nil {
		h.logger.Warn("failed to get peer profile", zap.Error(err), zap.Uint("peer_id", cmd.PeerID))
		return DirectChatResult{}, err
	}

	obj, err := h.manager.OpenDirectChat(ctx, services.ChatCreateParams{
		UserID: cmd.UserID,
		PeerID: cmd.PeerID,
	})
	if err != nil {
		h.logger.Warn("failed to open direct chat", zap.Error(err),
			zap.Uint("user_id", cmd.UserID), zap.Uint("peer_id", cmd.PeerID))
		return DirectChatResult{}, err
	}

	// Синхронизируем комнаты: онлайн-сокеты обоих участников заводим в chat:<id>,
	// чтобы они сразу получали события чата. Best-effort — ошибка не влияет на
	// ответ (участники доберут события через историю при следующем подключении).
	if err := h.rooms.JoinUsers(ctx, obj.ID, []uint{cmd.UserID, cmd.PeerID}); err != nil {
		h.logger.Warn("failed to sync direct chat rooms", zap.Error(err), zap.Uint("chat_id", obj.ID))
	}

	h.logger.Info("opened direct chat", zap.Uint("chat_id", obj.ID))
	return toDirectChatResult(obj, prof), nil
}
