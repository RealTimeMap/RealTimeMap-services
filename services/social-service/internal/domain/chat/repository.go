package chat

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, obj *Chat) error
	GetByID(ctx context.Context, id uint) (*Chat, error)
	// GetOrCreateDirect строит direct_key из (userA, userB) и дедуплицирует
	// чат на уровне БД через ON CONFLICT. Участники создаются в той же транзакции.
	GetOrCreateDirect(ctx context.Context, userA, userB uint) (*Chat, error)
	// ListByUser возвращает обогащённую проекцию (чат + last message + unread).
	ListByUser(ctx context.Context, userID uint) ([]*ChatListItem, error)
	// UpdateLastMessage обновляет денормализованный указатель на последнее
	// сообщение — для сортировки списка чатов и превью.
	UpdateLastMessage(ctx context.Context, chatID, messageID uint) error
}

type ParticipantRepository interface {
	Add(ctx context.Context, obj *ChatParticipant) error
	BulkAdd(ctx context.Context, objs []*ChatParticipant) error
	Get(ctx context.Context, chatID, userID uint) (*ChatParticipant, error)
	ListByChat(ctx context.Context, chatID uint) ([]*ChatParticipant, error)
	// ChatIDsByUser возвращает id всех активных чатов пользователя (не вышедших).
	// Используется socket-слоем для eager-join в комнаты chat:<id> при подключении.
	ChatIDsByUser(ctx context.Context, userID uint) ([]uint, error)
	// PeerIDsByUser возвращает id пользователей, с которыми userID состоит хотя бы
	// в одном активном чате. Используется socket-слоем для presence-снапшота:
	// подключившемуся надо сразу показать, кто из его собеседников уже онлайн.
	PeerIDsByUser(ctx context.Context, userID uint) ([]uint, error)
	//UpdateRole(ctx context.Context)
	UpdateLastRead(ctx context.Context, chatID, userID, lastReadID uint) error
	Remove(ctx context.Context, chatID, userID uint) error
}
