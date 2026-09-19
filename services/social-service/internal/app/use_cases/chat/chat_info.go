package chat

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
)

// ChatGetter — доступ к чату по id. *services.ChatService удовлетворяет
// интерфейсу напрямую.
type ChatGetter interface {
	GetChat(ctx context.Context, chatID uint) (*chat.Chat, error)
}

// chatInfoAdapter достаёт из чата ровно то, что нужно тексту уведомления.
//
// Адаптер, а не прямая зависимость от ChatService: потребителю нужны два поля,
// и тянуть ради них доменную модель в контракт события незачем.
type chatInfoAdapter struct {
	chats ChatGetter
}

// NewChatInfoGetter собирает источник данных о чате поверх ChatGetter.
func NewChatInfoGetter(chats ChatGetter) ChatInfoGetter {
	if chats == nil {
		return noopChatInfo{}
	}
	return &chatInfoAdapter{chats: chats}
}

func (a *chatInfoAdapter) ChatInfo(ctx context.Context, chatID uint) (string, bool, error) {
	obj, err := a.chats.GetChat(ctx, chatID)
	if err != nil {
		return "", false, err
	}
	if obj == nil {
		return "", false, nil
	}

	var title string
	if obj.Title != nil {
		title = *obj.Title
	}

	return title, obj.Type == chat.GroupType, nil
}
