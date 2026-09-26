package postgres

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/dbtest"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Интеграционные тесты удаления чатов: доменные правила вместе с SQL списка,
// непрочитанных и истории на настоящем Postgres.
//
//	go test ./services/social-service/... -run Integration -v

type chatEnv struct {
	db       *gorm.DB
	chats    *services.ChatService
	messages *services.MessageService
	chatRepo chat.Repository
}

func newChatEnv(t *testing.T) *chatEnv {
	t.Helper()
	db := dbtest.Open(t, "rtm_social")
	require.NoError(t, db.AutoMigrate(&model.BlockedUser{}, &chat.Chat{}, &chat.ChatParticipant{}, &message.Message{}))

	log := zap.NewNop()
	txm := txmanager.NewTxManager(db)
	chatRepo := NewChatRepository(db, log)
	partRepo := NewChatParticipantRepository(db, log)
	blocked := NewPgBlockedUserRepository(db, log)

	return &chatEnv{
		db:       db,
		chats:    services.NewChatService(chatRepo, partRepo, blocked, &txm, log),
		messages: services.NewMessageService(NewMessageRepository(db, log), chatRepo, partRepo, blocked, &txm, log),
		chatRepo: chatRepo,
	}
}

// users выдаёт идентификаторы из диапазона, которого нет в dev-данных, и
// убирает за тестом все чаты этих пользователей.
func (e *chatEnv) users(t *testing.T, n int) []uint {
	base := uint(900_000_000 + rand.IntN(1_000_000)*10)
	ids := make([]uint, n)
	for i := range ids {
		ids[i] = base + uint(i) + 1
	}
	t.Cleanup(func() {
		var byMember, byCreator []uint
		e.db.Model(&chat.ChatParticipant{}).Where("user_id IN ?", ids).Pluck("chat_id", &byMember)
		e.db.Unscoped().Model(&chat.Chat{}).Where("created_by IN ?", ids).Pluck("id", &byCreator)
		chatIDs := append(byMember, byCreator...)
		if len(chatIDs) > 0 {
			e.db.Unscoped().Where("chat_id IN ?", chatIDs).Delete(&message.Message{})
			e.db.Where("chat_id IN ?", chatIDs).Delete(&chat.ChatParticipant{})
			e.db.Unscoped().Where("id IN ?", chatIDs).Delete(&chat.Chat{})
		}
	})
	return ids
}

func (e *chatEnv) send(t *testing.T, chatID, from uint, text string) {
	t.Helper()
	_, err := e.messages.SendMessage(context.Background(), services.MessageCreateParams{
		ChatID: chatID, SenderID: from, Content: text,
	})
	require.NoError(t, err)
}

// listed сообщает, виден ли чат в списке пользователя, и сколько в нём
// непрочитанных.
func (e *chatEnv) listed(t *testing.T, userID, chatID uint) (bool, int) {
	t.Helper()
	items, err := e.chatRepo.ListByUser(context.Background(), userID)
	require.NoError(t, err)
	for _, it := range items {
		if it.Chat.ID == chatID {
			return true, it.UnreadCount
		}
	}
	return false, 0
}

func (e *chatEnv) history(t *testing.T, chatID, userID uint) []string {
	t.Helper()
	msgs, err := e.messages.History(context.Background(), services.MessageGetParams{ChatID: chatID, UserID: userID})
	require.NoError(t, err)
	texts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		texts = append(texts, m.Content)
	}
	return texts
}

func TestIntegrationDeleteDirectForMe(t *testing.T) {
	e := newChatEnv(t)
	ctx := context.Background()
	u := e.users(t, 2)
	me, peer := u[0], u[1]

	c, err := e.chats.OpenDirectChat(ctx, services.ChatCreateParams{UserID: me, PeerID: peer})
	require.NoError(t, err)
	e.send(t, c.ID, peer, "старое 1")
	e.send(t, c.ID, me, "старое 2")

	res, err := e.chats.Delete(ctx, c.ID, me, false)
	require.NoError(t, err)
	assert.False(t, res.ForEveryone)
	assert.Equal(t, []uint{me}, res.ParticipantIDs, "уведомляются только устройства удалившего")

	visible, _ := e.listed(t, me, c.ID)
	assert.False(t, visible, "чат скрыт у удалившего")
	assert.Empty(t, e.history(t, c.ID, me))

	visible, _ = e.listed(t, peer, c.ID)
	assert.True(t, visible, "у собеседника чат на месте")
	assert.Equal(t, []string{"старое 2", "старое 1"}, e.history(t, c.ID, peer))

	// Новое сообщение возвращает чат, но без старой переписки.
	e.send(t, c.ID, peer, "новое")
	visible, unread := e.listed(t, me, c.ID)
	assert.True(t, visible)
	assert.Equal(t, 1, unread, "удалённые сообщения не считаются непрочитанными")
	assert.Equal(t, []string{"новое"}, e.history(t, c.ID, me))
}

func TestIntegrationDeleteEmptyDirectForMe(t *testing.T) {
	e := newChatEnv(t)
	ctx := context.Background()
	u := e.users(t, 2)

	c, err := e.chats.OpenDirectChat(ctx, services.ChatCreateParams{UserID: u[0], PeerID: u[1]})
	require.NoError(t, err)

	_, err = e.chats.Delete(ctx, c.ID, u[0], false)
	require.NoError(t, err)
	visible, _ := e.listed(t, u[0], c.ID)
	assert.False(t, visible, "чат без сообщений тоже скрывается")

	e.send(t, c.ID, u[1], "привет")
	visible, _ = e.listed(t, u[0], c.ID)
	assert.True(t, visible)
}

func TestIntegrationDeleteDirectForEveryone(t *testing.T) {
	e := newChatEnv(t)
	ctx := context.Background()
	u := e.users(t, 2)

	c, err := e.chats.OpenDirectChat(ctx, services.ChatCreateParams{UserID: u[0], PeerID: u[1]})
	require.NoError(t, err)
	e.send(t, c.ID, u[1], "привет")
	var msgID uint
	require.NoError(t, e.db.Model(&message.Message{}).Where("chat_id = ?", c.ID).Pluck("id", &msgID).Error)
	require.NoError(t, e.db.Delete(&message.Message{}, msgID).Error, "мягко удалённое тоже должно уйти")
	e.send(t, c.ID, u[0], "ответ")

	res, err := e.chats.Delete(ctx, c.ID, u[0], true)
	require.NoError(t, err)
	assert.True(t, res.ForEveryone)
	assert.ElementsMatch(t, u, res.ParticipantIDs)

	var n int64
	e.db.Unscoped().Model(&chat.Chat{}).Where("id = ?", c.ID).Count(&n)
	assert.Zero(t, n, "чат удалён физически, а не мягко")
	e.db.Model(&chat.ChatParticipant{}).Where("chat_id = ?", c.ID).Count(&n)
	assert.Zero(t, n, "участники")
	e.db.Unscoped().Model(&message.Message{}).Where("chat_id = ?", c.ID).Count(&n)
	assert.Zero(t, n, "сообщения, включая мягко удалённые")

	// Повторное открытие создаёт новый пустой чат.
	again, err := e.chats.OpenDirectChat(ctx, services.ChatCreateParams{UserID: u[1], PeerID: u[0]})
	require.NoError(t, err)
	assert.NotEqual(t, c.ID, again.ID)
	assert.Empty(t, e.history(t, again.ID, u[0]))
}

func TestIntegrationDeleteGroup(t *testing.T) {
	e := newChatEnv(t)
	ctx := context.Background()
	u := e.users(t, 4)
	owner, member, outsider := u[0], u[1], u[3]

	g, err := e.chats.OpenGroupChat(ctx, services.GroupChatCreateParams{OwnerID: owner, PeersIds: []uint{member, u[2]}})
	require.NoError(t, err)
	e.send(t, g.ID, member, "всем привет")

	_, err = e.chats.Delete(ctx, g.ID, member, true)
	assertForbidden(t, err, "участник не может удалить группу")

	_, err = e.chats.Delete(ctx, g.ID, outsider, false)
	assertForbidden(t, err, "не участник не может удалить чат")

	// Для группы флаг не важен: владелец удаляет её у всех.
	res, err := e.chats.Delete(ctx, g.ID, owner, false)
	require.NoError(t, err)
	assert.True(t, res.ForEveryone)
	assert.ElementsMatch(t, u[:3], res.ParticipantIDs)

	for _, id := range u[:3] {
		visible, _ := e.listed(t, id, g.ID)
		assert.False(t, visible)
	}
}

func TestIntegrationDeleteMissingChat(t *testing.T) {
	e := newChatEnv(t)
	_, err := e.chats.Delete(context.Background(), 4_000_000_000, 1, false)
	var notFound *apperror.NotFoundError
	assert.ErrorAs(t, err, &notFound)
}

func assertForbidden(t *testing.T, err error, msg string) {
	t.Helper()
	var forbidden *apperror.ForbiddenError
	assert.ErrorAs(t, err, &forbidden, msg)
}
