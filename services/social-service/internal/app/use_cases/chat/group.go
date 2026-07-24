package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"go.uber.org/zap"
)

type GroupManager interface {
	OpenGroupChat(ctx context.Context, param services.GroupChatCreateParams) (*chat.Chat, error)
}

type GroupChatHandler struct {
	manager GroupManager
	rooms   EventPublisher

	logger *zap.Logger
}

func NewGroupChatHandler(manager GroupManager, rooms EventPublisher, logger *zap.Logger) *GroupChatHandler {
	return &GroupChatHandler{
		manager: manager,
		rooms:   rooms,
		logger:  logger,
	}
}

type GroupChatCreateCommand struct {
	InitiatorID uint
	PeersIds    []uint
	Title       *string
}

func (c GroupChatCreateCommand) validate() error {
	ids := make([]uint, 0, len(c.PeersIds)+1)
	ids = append(ids, c.PeersIds...)
	ids = append(ids, c.InitiatorID)

	if !utils.IsUniqueElement(ids) {
		return chat.ErrDuplicateMembers(ids)
	}
	return nil
}

func (h *GroupChatHandler) Handle(ctx context.Context, cmd GroupChatCreateCommand) (GroupChatResult, error) {
	h.logger.Info("start chatUseCases.GroupChatHandler.Handle",
		zap.Uint("initiator_id", cmd.InitiatorID), zap.Int("peers", len(cmd.PeersIds)))

	if err := cmd.validate(); err != nil {
		h.logger.Warn("invalid group chat command", zap.Error(err))
		return GroupChatResult{}, err
	}

	obj, err := h.manager.OpenGroupChat(ctx, services.GroupChatCreateParams{
		OwnerID:  cmd.InitiatorID,
		PeersIds: cmd.PeersIds,
		Title:    cmd.Title,
	})
	if err != nil {
		h.logger.Warn("failed to open group chat", zap.Error(err))
		return GroupChatResult{}, err
	}

	// Синхронизируем комнаты: онлайн-сокеты всех участников (инициатор + peers)
	// заводим в chat:<id>. Best-effort — ошибка не влияет на ответ.
	memberIDs := make([]uint, 0, len(cmd.PeersIds)+1)
	memberIDs = append(memberIDs, cmd.InitiatorID)
	memberIDs = append(memberIDs, cmd.PeersIds...)
	if err := h.rooms.JoinUsers(ctx, obj.ID, memberIDs); err != nil {
		h.logger.Warn("failed to sync group chat rooms", zap.Error(err), zap.Uint("chat_id", obj.ID))
	}

	h.logger.Info("opened group chat", zap.Uint("chat_id", obj.ID))

	return toGroupChatResult(obj), nil
}
