package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChatRepository struct {
	db *gorm.DB

	log *zap.Logger
}

func NewChatRepository(db *gorm.DB, log *zap.Logger) chat.Repository {
	return &ChatRepository{
		db:  db,
		log: log,
	}
}

// dbCtx возвращает транзакцию из контекста (если сервис обернул вызов в
// txmanager.WithTx) либо собственный пул. Так репозиторий участвует в общей
// транзакции — например при атомарном создании группового чата.
func (r *ChatRepository) dbCtx(ctx context.Context) *gorm.DB {
	return txmanager.DBFromCtx(ctx, r.db)
}

func (r *ChatRepository) Create(ctx context.Context, obj *chat.Chat) error {
	r.log.Info("Create Chat", zap.Any("chat", obj))
	err := r.dbCtx(ctx).Create(obj).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *ChatRepository) GetByID(ctx context.Context, id uint) (*chat.Chat, error) {
	r.log.Info("Get Chat", zap.Uint("chat_id", id))
	var obj chat.Chat
	err := r.dbCtx(ctx).
		Preload("Participants").
		First(&obj, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, chat.ErrChatNotFound(id)
		}
		return nil, err
	}
	return &obj, nil
}

// GetOrCreateDirect атомарно возвращает существующий личный чат либо создаёт новый.
// Дедупликация — на уровне БД через уникальный direct_key + ON CONFLICT DO NOTHING,
// что закрывает гонку при параллельном создании. Участники создаются в той же транзакции.
func (r *ChatRepository) GetOrCreateDirect(ctx context.Context, userA, userB uint) (*chat.Chat, error) {
	if userA == userB {
		return nil, chat.ErrSelfDirectChat(userA)
	}

	key := directKey(userA, userB)
	var result chat.Chat

	err := r.dbCtx(ctx).Transaction(func(tx *gorm.DB) error {
		newChat := chat.Chat{
			Type:      chat.DirectType,
			DirectKey: &key,
			CreatedBy: userA,
		}

		// ON CONFLICT по direct_key: при гонке вторая вставка ничего не сделает.
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "direct_key"}},
			DoNothing: true,
		}).Create(&newChat).Error; err != nil {
			return err
		}

		// ID == 0 → строка не вставилась (конфликт), чат уже существовал — дочитываем.
		if newChat.ID == 0 {
			return tx.Preload("Participants").
				Where("direct_key = ?", key).
				First(&result).Error
		}

		participants := []chat.ChatParticipant{
			{ChatID: newChat.ID, UserID: userA, Role: chat.MemberParticipantType},
			{ChatID: newChat.ID, UserID: userB, Role: chat.MemberParticipantType},
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&participants).Error; err != nil {
			return err
		}

		newChat.Participants = participants
		result = newChat
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// ListByUser возвращает чаты пользователя вместе с превью последнего сообщения
// и числом непрочитанных, отсортированные по свежести. Собирается без N+1:
// список чатов — один запрос, затем batch-подгрузка сообщений и unread.
func (r *ChatRepository) ListByUser(ctx context.Context, userID uint) ([]*chat.ChatListItem, error) {
	// 1. Чаты, где пользователь — активный участник (не вышел).
	// Preload участников нужен, чтобы на уровне use-case определить собеседника
	// в direct-чатах (имя/аватар чата = профиль второго участника).
	var chats []chat.Chat
	err := r.dbCtx(ctx).
		Preload("Participants").
		Joins("JOIN chat_participants cp ON cp.chat_id = chats.id").
		Where("cp.user_id = ? AND cp.left_at IS NULL", userID).
		Order("chats.last_message_id DESC NULLS LAST").
		Find(&chats).Error
	if err != nil {
		return nil, err
	}
	if len(chats) == 0 {
		return []*chat.ChatListItem{}, nil
	}

	chatIDs := make([]uint, 0, len(chats))
	lastMsgIDs := make([]uint, 0, len(chats))
	for i := range chats {
		chatIDs = append(chatIDs, chats[i].ID)
		if chats[i].LastMessageID != nil {
			lastMsgIDs = append(lastMsgIDs, *chats[i].LastMessageID)
		}
	}

	// 2. Batch-подгрузка превью последних сообщений.
	lastMessages := make(map[uint]*message.Message, len(lastMsgIDs))
	if len(lastMsgIDs) > 0 {
		var msgs []message.Message
		if err = r.dbCtx(ctx).
			Where("id IN ?", lastMsgIDs).
			Find(&msgs).Error; err != nil {
			return nil, err
		}
		for i := range msgs {
			lastMessages[msgs[i].ID] = &msgs[i]
		}
	}

	// 3. Batch-подсчёт непрочитанных для всех чатов одним запросом.
	unread, err := r.unreadCounts(ctx, userID, chatIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*chat.ChatListItem, 0, len(chats))
	for i := range chats {
		item := &chat.ChatListItem{
			Chat:        chats[i],
			UnreadCount: unread[chats[i].ID],
		}
		if chats[i].LastMessageID != nil {
			item.LastMessage = lastMessages[*chats[i].LastMessageID]
		}
		items = append(items, item)
	}
	return items, nil
}

// unreadCounts считает непрочитанные сообщения по каждому чату для пользователя:
// сообщения с id > last_read_message_id участника и не им самим отправленные.
func (r *ChatRepository) unreadCounts(ctx context.Context, userID uint, chatIDs []uint) (map[uint]int, error) {
	type row struct {
		ChatID uint
		Cnt    int
	}
	var rows []row

	err := r.dbCtx(ctx).
		Model(&message.Message{}).
		Select("messages.chat_id AS chat_id, COUNT(*) AS cnt").
		Joins("JOIN chat_participants cp ON cp.chat_id = messages.chat_id AND cp.user_id = ?", userID).
		Where("messages.chat_id IN ?", chatIDs).
		Where("messages.sender_id <> ?", userID).
		Where("cp.last_read_message_id IS NULL OR messages.id > cp.last_read_message_id").
		Group("messages.chat_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uint]int, len(rows))
	for _, rw := range rows {
		counts[rw.ChatID] = rw.Cnt
	}
	return counts, nil
}

// UpdateLastMessage обновляет денормализованный указатель на последнее сообщение чата.
func (r *ChatRepository) UpdateLastMessage(ctx context.Context, chatID, messageID uint) error {
	return r.dbCtx(ctx).
		Model(&chat.Chat{}).
		Where("id = ?", chatID).
		Update("last_message_id", messageID).Error
}

func directKey(a, b uint) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("%d_%d", a, b)
}
