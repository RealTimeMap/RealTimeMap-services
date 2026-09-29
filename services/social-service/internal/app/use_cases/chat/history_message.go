package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"go.uber.org/zap"
)

type HistoryGetter interface {
	History(ctx context.Context, params services.MessageGetParams) ([]*message.Message, error)
	// ReadCursors — курсоры прочтения активных участников чата.
	ReadCursors(ctx context.Context, chatID uint) ([]chat.ReadCursor, error)
}

type ChatHistoryHandler struct {
	getter         HistoryGetter
	profilesGetter ProfilesBatchGetter

	logger *zap.Logger
}

func NewChatHistoryHandler(getter HistoryGetter, profilesGetter ProfilesBatchGetter, logger *zap.Logger) *ChatHistoryHandler {
	return &ChatHistoryHandler{
		getter:         getter,
		profilesGetter: profilesGetter,
		logger:         logger,
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

	// History уже проверила, что пользователь — участник чата.
	cursors, err := h.getter.ReadCursors(ctx, cmd.ChatID)
	if err != nil {
		h.logger.Warn("failed to get read cursors", zap.Error(err), zap.Uint("chat_id", cmd.ChatID))
		return MessageHistoryResult{}, err
	}

	profiles := h.loadSenderProfiles(ctx, messages)

	h.logger.Info("chat history fetched",
		zap.Uint("chat_id", cmd.ChatID), zap.Int("count", len(messages)))
	return toMessageHistoryResult(messages, profiles, nextCursor(messages), cursors), nil
}

// loadSenderProfiles одним запросом подгружает профили авторов. Best-effort.
func (h *ChatHistoryHandler) loadSenderProfiles(ctx context.Context, messages []*message.Message) map[uint]*model.Profile {
	ids := make([]uint, 0, len(messages))
	for _, m := range messages {
		ids = append(ids, m.SenderID)
	}
	ids = utils.UniqueValues(ids)

	profiles := make(map[uint]*model.Profile, len(ids))
	if len(ids) == 0 {
		return profiles
	}

	list, err := h.profilesGetter.GetProfilesByIDs(ctx, ids)
	if err != nil {
		h.logger.Warn("failed to batch-load sender profiles", zap.Error(err))
		return profiles
	}
	for _, p := range list {
		profiles[p.UserID] = p
	}
	return profiles
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
