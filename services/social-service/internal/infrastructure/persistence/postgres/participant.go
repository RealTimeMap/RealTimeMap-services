package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChatParticipantRepository struct {
	db *gorm.DB

	log *zap.Logger
}

func NewChatParticipantRepository(db *gorm.DB, log *zap.Logger) chat.ParticipantRepository {
	return &ChatParticipantRepository{
		db:  db,
		log: log,
	}
}

// dbCtx возвращает транзакцию из контекста (если сервис обернул вызов в
// txmanager.WithTx) либо собственный пул. Так репозиторий участвует в общей
// транзакции — например при атомарном создании группового чата.
func (r *ChatParticipantRepository) dbCtx(ctx context.Context) *gorm.DB {
	return txmanager.DBFromCtx(ctx, r.db)
}

// Add добавляет участника в чат. Идемпотентно: повторное добавление того же
// (chat_id, user_id) не создаёт дубликат и не является ошибкой.
func (r *ChatParticipantRepository) Add(ctx context.Context, obj *chat.ChatParticipant) error {
	return r.dbCtx(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(obj).Error
}

// Get возвращает участника чата. nil без ошибки — если пользователь не участник.
func (r *ChatParticipantRepository) Get(ctx context.Context, chatID, userID uint) (*chat.ChatParticipant, error) {
	var p chat.ChatParticipant
	err := r.dbCtx(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// ListByChat возвращает активных участников чата (не вышедших).
func (r *ChatParticipantRepository) ListByChat(ctx context.Context, chatID uint) ([]*chat.ChatParticipant, error) {
	var participants []*chat.ChatParticipant
	err := r.dbCtx(ctx).
		Where("chat_id = ? AND left_at IS NULL", chatID).
		Find(&participants).Error
	if err != nil {
		return nil, err
	}
	return participants, nil
}

// ChatIDsByUser возвращает id активных чатов пользователя (left_at IS NULL).
// Лёгкая проекция только id — для eager-join сокета в комнаты chat:<id>.
func (r *ChatParticipantRepository) ChatIDsByUser(ctx context.Context, userID uint) ([]uint, error) {
	var ids []uint
	err := r.dbCtx(ctx).
		Model(&chat.ChatParticipant{}).
		Where("user_id = ? AND left_at IS NULL", userID).
		Pluck("chat_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// PeerIDsByUser возвращает id пользователей, с которыми userID состоит хотя бы в
// одном активном чате (сам userID исключён). Один запрос с самосоединением по
// chat_id вместо «список чатов → участники каждого»: presence-снапшот строится на
// подключении сокета, лишние round-trip-ы здесь нежелательны.
func (r *ChatParticipantRepository) PeerIDsByUser(ctx context.Context, userID uint) ([]uint, error) {
	var ids []uint
	err := r.dbCtx(ctx).
		Model(&chat.ChatParticipant{}).
		Distinct("peers.user_id").
		Joins("JOIN chat_participants AS peers ON peers.chat_id = chat_participants.chat_id AND peers.left_at IS NULL").
		Where("chat_participants.user_id = ? AND chat_participants.left_at IS NULL", userID).
		Where("peers.user_id <> ?", userID).
		Pluck("peers.user_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// UpdateLastRead двигает курсор прочитанного вперёд. Курсор монотонный:
// значение не откатывается назад, если пришёл меньший last_read_id (out-of-order запросы).
func (r *ChatParticipantRepository) UpdateLastRead(ctx context.Context, chatID, userID, lastReadID uint) error {
	return r.dbCtx(ctx).
		Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Where("last_read_message_id IS NULL OR last_read_message_id < ?", lastReadID).
		Update("last_read_message_id", lastReadID).Error
}

// Remove помечает участника вышедшим (soft-leave через left_at), сохраняя историю
// членства. Физически строку не удаляем.
func (r *ChatParticipantRepository) Remove(ctx context.Context, chatID, userID uint) error {
	return r.dbCtx(ctx).
		Model(&chat.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ? AND left_at IS NULL", chatID, userID).
		Update("left_at", gorm.Expr("now()")).Error
}

func (r *ChatParticipantRepository) BulkAdd(ctx context.Context, objs []*chat.ChatParticipant) error {
	r.log.Info("BulkAdd called", zap.Int("objs", len(objs)))
	err := r.dbCtx(ctx).Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(objs).Error
	if err != nil {
		return err
	}
	return nil
}
