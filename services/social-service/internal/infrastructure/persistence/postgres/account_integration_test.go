package postgres

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/dbtest"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Интеграционный тест стирания данных: проверяет SQL на настоящем Postgres —
// что удалено, что обезличено и что чужие данные не задеты.
//
//	go test ./services/social-service/... -run Integration -v

func TestIntegrationEraseUser(t *testing.T) {
	db := dbtest.Open(t, "rtm_social")
	require.NoError(t, db.AutoMigrate(
		&model.Profile{}, &model.Friendship{}, &model.BlockedUser{}, &model.Subscription{},
		&chat.Chat{}, &chat.ChatParticipant{}, &message.Message{},
	))
	ctx := context.Background()

	// Идентификаторы из диапазона, которого нет в dev-данных.
	base := uint(900_000_000 + rand.IntN(1_000_000)*10)
	victim, friend, stranger := base+1, base+2, base+3

	clientID := "client-msg-1"
	require.NoError(t, db.Create(&[]model.Profile{
		{UserID: victim, Username: "victim", Tag: tag(victim)},
		{UserID: friend, Username: "friend", Tag: tag(friend)},
		{UserID: stranger, Username: "stranger", Tag: tag(stranger)},
	}).Error)
	require.NoError(t, db.Create(&[]model.Friendship{
		{UserID: victim, FriendID: friend}, {UserID: friend, FriendID: victim},
		{UserID: friend, FriendID: stranger},
	}).Error)
	require.NoError(t, db.Create(&[]model.Subscription{
		{SubscriberID: stranger, TargetID: victim},
		{SubscriberID: friend, TargetID: stranger},
	}).Error)
	require.NoError(t, db.Create(&model.BlockedUser{UserID: victim, BlockedUserID: stranger}).Error)

	c := chat.Chat{CreatedBy: victim}
	require.NoError(t, db.Create(&c).Error)
	t.Cleanup(func() { cleanupSocial(db, c.ID, victim, friend, stranger) })
	require.NoError(t, db.Create(&[]chat.ChatParticipant{
		{ChatID: c.ID, UserID: victim}, {ChatID: c.ID, UserID: friend},
	}).Error)
	require.NoError(t, db.Create(&[]message.Message{
		{ChatID: c.ID, SenderID: victim, Content: "hi", ClientMessageID: &clientID},
		{ChatID: c.ID, SenderID: friend, Content: "hello"},
	}).Error)

	eraser := NewPgAccountEraser(db)
	affected, err := eraser.EraseUser(ctx, victim)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint{friend, stranger}, affected)

	assert.Zero(t, count(t, db, &model.Profile{}, "user_id = ?", victim))
	assert.Zero(t, count(t, db, &model.Friendship{}, "user_id = ? OR friend_id = ?", victim, victim))
	assert.Zero(t, count(t, db, &model.Subscription{}, "subscriber_id = ? OR target_id = ?", victim, victim))
	assert.Zero(t, count(t, db, &model.BlockedUser{}, "user_id = ? OR blocked_user_id = ?", victim, victim))
	assert.Zero(t, count(t, db, &chat.ChatParticipant{}, "user_id = ?", victim))
	assert.Zero(t, count(t, db, &message.Message{}, "sender_id = ?", victim))

	// Чужие данные на месте, сообщения удалённого остались в чате обезличенными.
	assert.EqualValues(t, 1, count(t, db, &model.Profile{}, "user_id = ?", friend))
	assert.EqualValues(t, 1, count(t, db, &model.Friendship{}, "user_id = ? AND friend_id = ?", friend, stranger))
	assert.EqualValues(t, 1, count(t, db, &model.Subscription{}, "subscriber_id = ? AND target_id = ?", friend, stranger))
	assert.EqualValues(t, 2, count(t, db, &message.Message{}, "chat_id = ?", c.ID))
	assert.EqualValues(t, 1, count(t, db, &message.Message{},
		"chat_id = ? AND sender_id = 0 AND client_message_id IS NULL", c.ID))
	assert.EqualValues(t, 1, count(t, db, &chat.Chat{}, "id = ? AND created_by = 0", c.ID))

	// Повторная доставка события ничего не ломает.
	affected, err = eraser.EraseUser(ctx, victim)
	require.NoError(t, err)
	assert.Empty(t, affected)
}

func count(t *testing.T, db *gorm.DB, m any, where string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Unscoped().Model(m).Where(where, args...).Count(&n).Error)
	return n
}

func tag(id uint) string { return fmt.Sprintf("test%d", id) }

func cleanupSocial(db *gorm.DB, chatID uint, ids ...uint) {
	db.Unscoped().Where("chat_id = ?", chatID).Delete(&message.Message{})
	db.Where("chat_id = ?", chatID).Delete(&chat.ChatParticipant{})
	db.Unscoped().Delete(&chat.Chat{}, chatID)
	db.Where("user_id IN ? OR friend_id IN ?", ids, ids).Delete(&model.Friendship{})
	db.Where("subscriber_id IN ? OR target_id IN ?", ids, ids).Delete(&model.Subscription{})
	db.Where("user_id IN ? OR blocked_user_id IN ?", ids, ids).Delete(&model.BlockedUser{})
	db.Where("user_id IN ?", ids).Delete(&model.Profile{})
}
