package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"go.uber.org/zap"
)

type HistoryGetter interface {
	History(ctx context.Context, params services.MessageGetParams) ([]*message.Message, error)
}

type ChatHistoryHandler struct {
	getter        HistoryGetter
	profileGetter ProfileGetter

	logger *zap.Logger
}

func NewChatHistoryHandler(getter HistoryGetter, profileGetter ProfileGetter, logger *zap.Logger) *ChatHistoryHandler {
	return &ChatHistoryHandler{
		getter:        getter,
		profileGetter: profileGetter,
		logger:        logger,
	}
}

type GetMessageCommand struct {
	UserID        uint
	ChatID        uint
	LastMessageID *uint
}

func (h *ChatHistoryHandler) Handle(ctx context.Context, cmd GetMessageCommand) (MessageHistoryResult, error) {
	h.logger.Info("start chatUseCases.ChatHistoryHandler.Handle",
		zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))

	messages, err := h.getter.History(ctx, services.MessageGetParams{
		ChatID:        cmd.ChatID,
		UserID:        cmd.UserID,
		LastMessageID: cmd.LastMessageID,
	})
	if err != nil {
		h.logger.Warn("failed to get chat history", zap.Error(err),
			zap.Uint("chat_id", cmd.ChatID), zap.Uint("user_id", cmd.UserID))
		return MessageHistoryResult{}, err
	}

	userIds := make([]uint, 0, len(messages))
	for _, m := range messages {
		userIds = append(userIds, m.SenderID)
	}
	userIds = utils.UniqueValues(userIds)

	profiles := make(map[uint]*model.Profile, len(userIds))
	for _, id := range userIds {
		p, err := h.profileGetter.GetProfile(ctx, id)
		if err != nil {
			h.logger.Warn("failed to get profile", zap.Error(err), zap.Uint("user_id", id))
			continue
		}
		profiles[p.UserID] = p
	}

	h.logger.Info("chat history fetched",
		zap.Uint("chat_id", cmd.ChatID), zap.Int("count", len(messages)))
	return toMessageHistoryResult(messages, profiles, nextCursor(messages)), nil
}

// nextCursor возвращает id последнего (самого старого) сообщения страницы —
// курсор для запроса следующей, более старой порции через keyset-пагинацию.
// nil, если страница пуста.
func nextCursor(messages []*message.Message) *uint {
	if len(messages) == 0 {
		return nil
	}
	last := messages[len(messages)-1].ID
	return &last
}
