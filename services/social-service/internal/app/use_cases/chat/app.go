package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
)

type Application struct {
	Direct      *DirectChatHandler
	Group       *GroupChatHandler
	SendMessage *MessageSenderHandler
	History     *ChatHistoryHandler
	ListChats   *ListUserChatsHandler
	MarkRead    *MarkReadHandler
	Leave       *LeaveHandler
}

type ProfileGetter interface {
	GetProfile(ctx context.Context, userId uint) (*model.Profile, error) // TODO Поменять импорт на новый паттерн как пененесется та часть приложения
}
