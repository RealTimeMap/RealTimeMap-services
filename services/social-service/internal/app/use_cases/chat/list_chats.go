package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"go.uber.org/zap"
)

type ChatsLister interface {
	GetUsersChats(ctx context.Context, userID uint) ([]*chat.ChatListItem, error)
}

// ProfilesBatchGetter отдаёт профили пачкой
type ProfilesBatchGetter interface {
	GetProfilesByIDs(ctx context.Context, ids []uint) ([]*model.Profile, error)
}

type ListUserChatsHandler struct {
	lister         ChatsLister
	profilesGetter ProfilesBatchGetter

	logger *zap.Logger
}

func NewListUserChatsHandler(lister ChatsLister, profilesGetter ProfilesBatchGetter, logger *zap.Logger) *ListUserChatsHandler {
	return &ListUserChatsHandler{
		lister:         lister,
		profilesGetter: profilesGetter,
		logger:         logger,
	}
}

type ListUserChatsCommand struct {
	UserID uint
}

func (h *ListUserChatsHandler) Handle(ctx context.Context, cmd ListUserChatsCommand) ([]ChatListItemResult, error) {
	h.logger.Info("start chatUseCases.ListUserChatsHandler.Handle", zap.Uint("user_id", cmd.UserID))

	items, err := h.lister.GetUsersChats(ctx, cmd.UserID)
	if err != nil {
		h.logger.Warn("failed to list user chats", zap.Error(err), zap.Uint("user_id", cmd.UserID))
		return nil, err
	}
	if len(items) == 0 {
		return []ChatListItemResult{}, nil
	}

	profiles, err := h.loadProfiles(ctx, items, cmd.UserID)
	if err != nil {
		return nil, err
	}

	h.logger.Info("user chats listed", zap.Uint("user_id", cmd.UserID), zap.Int("count", len(items)))
	return toChatListResult(items, cmd.UserID, profiles), nil
}

// loadProfiles одним батчем подгружает все профили, нужные для обогащения списка
func (h *ListUserChatsHandler) loadProfiles(ctx context.Context, items []*chat.ChatListItem, requesterID uint) (map[uint]*model.Profile, error) {
	ids := make([]uint, 0, len(items)*2)
	for _, item := range items {
		if item.Chat.Type == chat.DirectType {
			for _, p := range item.Chat.Participants {
				if p.UserID != requesterID {
					ids = append(ids, p.UserID)
				}
			}
		}
		if item.LastMessage != nil {
			ids = append(ids, item.LastMessage.SenderID)
		}
	}
	ids = utils.UniqueValues(ids)

	profiles := make(map[uint]*model.Profile, len(ids))
	if len(ids) == 0 {
		return profiles, nil
	}

	list, err := h.profilesGetter.GetProfilesByIDs(ctx, ids)
	if err != nil {
		h.logger.Warn("failed to batch-load profiles for chat list", zap.Error(err))
		return nil, err
	}
	for _, p := range list {
		profiles[p.UserID] = p
	}
	return profiles, nil
}
