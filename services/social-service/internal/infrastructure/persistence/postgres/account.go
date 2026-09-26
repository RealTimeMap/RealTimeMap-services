package postgres

import (
	"context"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/account"
	"gorm.io/gorm"
)

// deletedUserID — автор обезличенного сообщения или чата.
//
// Настоящие user_id начинаются с 1, поэтому 0 ни с кем не совпадёт: ни
// правка, ни удаление такого сообщения от чужого имени невозможны.
const deletedUserID uint = 0

type PgAccountEraser struct {
	db *gorm.DB
}

func NewPgAccountEraser(db *gorm.DB) account.Eraser {
	return &PgAccountEraser{db: db}
}

func (r *PgAccountEraser) EraseUser(ctx context.Context, userID uint) ([]uint, error) {
	var affected []uint

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		related, err := relatedUsers(tx, userID)
		if err != nil {
			return err
		}
		affected = related

		deletes := []struct {
			model any
			where string
		}{
			{&model.Friendship{}, "user_id = @id OR friend_id = @id"},
			{&model.Subscription{}, "subscriber_id = @id OR target_id = @id"},
			{&model.BlockedUser{}, "user_id = @id OR blocked_user_id = @id"},
			{&chat.ChatParticipant{}, "user_id = @id"},
			{&model.Profile{}, "user_id = @id"},
		}
		for _, d := range deletes {
			if err := tx.Where(d.where, map[string]any{"id": userID}).Delete(d.model).Error; err != nil {
				return err
			}
		}

		// Unscoped: сообщения, удалённые мягко, всё ещё хранят sender_id.
		// client_message_id обнуляется, потому что входит в уникальный индекс
		// вместе с sender_id: у двух удалённых пользователей в одном чате
		// совпадение ключа дедупликации уронило бы UPDATE.
		if err := tx.Unscoped().
			Model(&message.Message{}).
			Where("sender_id = ?", userID).
			Updates(map[string]any{
				"sender_id":         deletedUserID,
				"client_message_id": nil,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&chat.Chat{}).
			Where("created_by = ?", userID).
			Update("created_by", deletedUserID).Error
	})
	if err != nil {
		return nil, err
	}

	return affected, nil
}

// relatedUsers — пользователи, у которых после удаления изменятся счётчики
// друзей, подписок или подписчиков.
func relatedUsers(tx *gorm.DB, userID uint) ([]uint, error) {
	var ids []uint
	err := tx.Raw(`
		SELECT friend_id FROM friendships WHERE user_id = @id
		UNION SELECT user_id FROM friendships WHERE friend_id = @id
		UNION SELECT target_id FROM subscriptions WHERE subscriber_id = @id
		UNION SELECT subscriber_id FROM subscriptions WHERE target_id = @id`,
		map[string]any{"id": userID}).
		Scan(&ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
