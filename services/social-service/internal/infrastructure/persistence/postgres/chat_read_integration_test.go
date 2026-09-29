package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func (e *chatEnv) sendID(t *testing.T, chatID, from uint, text string) uint {
	t.Helper()
	m, err := e.messages.SendMessage(context.Background(), services.MessageCreateParams{
		ChatID: chatID, SenderID: from, Content: text,
	})
	require.NoError(t, err)
	return m.ID
}

func TestIntegrationLastMessageNotRolledBack(t *testing.T) {
	e := newChatEnv(t)
	ctx := context.Background()
	u := e.users(t, 2)
	me, peer := u[0], u[1]

	c, err := e.chats.OpenDirectChat(ctx, services.ChatCreateParams{UserID: me, PeerID: peer})
	require.NoError(t, err)
	older := e.sendID(t, c.ID, peer, "первое")
	newer := e.sendID(t, c.ID, peer, "второе")

	// Опоздавшее обновление от более старого сообщения.
	require.NoError(t, e.chatRepo.UpdateLastMessage(ctx, c.ID, older))

	got, err := e.chatRepo.GetByID(ctx, c.ID)
	require.NoError(t, err)
	require.NotNil(t, got.LastMessageID)
	assert.Equal(t, newer, *got.LastMessageID)

	lastRead, moved, err := e.chats.MarkRead(ctx, c.ID, me)
	require.NoError(t, err)
	assert.True(t, moved)
	assert.Equal(t, newer, lastRead)

	_, unread := e.listed(t, me, c.ID)
	assert.Equal(t, 0, unread, "после прочтения непрочитанных нет")
}

func TestIntegrationMessageInsertJoinsTransaction(t *testing.T) {
	e := newChatEnv(t)
	ctx := context.Background()
	u := e.users(t, 2)

	c, err := e.chats.OpenDirectChat(ctx, services.ChatCreateParams{UserID: u[0], PeerID: u[1]})
	require.NoError(t, err)

	txm := txmanager.NewTxManager(e.db)
	repo := NewMessageRepository(e.db, zap.NewNop())
	errRollback := errors.New("rollback")

	obj := &message.Message{ChatID: c.ID, SenderID: u[0], Content: "откатится", Type: message.TextType}
	err = txm.WithTx(ctx, func(ctx context.Context) error {
		if _, err := repo.CreateIdempotent(ctx, obj); err != nil {
			return err
		}
		return errRollback
	})
	require.ErrorIs(t, err, errRollback)

	var count int64
	require.NoError(t, e.db.Model(&message.Message{}).Where("chat_id = ?", c.ID).Count(&count).Error)
	assert.Zero(t, count, "вставка сообщения откатывается вместе с транзакцией")
}

func TestIntegrationReadCursors(t *testing.T) {
	e := newChatEnv(t)
	ctx := context.Background()
	u := e.users(t, 3)
	owner, reader, leaver := u[0], u[1], u[2]

	c, err := e.chats.OpenGroupChat(ctx, services.GroupChatCreateParams{OwnerID: owner, PeersIds: []uint{reader, leaver}})
	require.NoError(t, err)
	last := e.sendID(t, c.ID, owner, "привет")

	_, _, err = e.chats.MarkRead(ctx, c.ID, reader)
	require.NoError(t, err)
	require.NoError(t, e.chats.Leave(ctx, c.ID, leaver))

	want := []chat.ReadCursor{
		{UserID: owner, LastReadMessageID: 0},
		{UserID: reader, LastReadMessageID: last},
	}

	cursors, err := e.messages.ReadCursors(ctx, c.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, want, cursors, "история: курсоры активных участников")

	items, err := e.chatRepo.ListByUser(ctx, owner)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.ElementsMatch(t, want, chat.ActiveReadCursors(items[0].Chat.Participants), "список чатов: те же курсоры")
}
