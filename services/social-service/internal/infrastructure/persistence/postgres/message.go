package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultMessagePageSize = 30

type MessageRepository struct {
	db *gorm.DB

	log *zap.Logger
}

func NewMessageRepository(db *gorm.DB, log *zap.Logger) message.Repository {
	return &MessageRepository{
		db:  db,
		log: log,
	}
}

func (r *MessageRepository) Create(ctx context.Context, obj *message.Message) error {
	return r.db.WithContext(ctx).Create(obj).Error
}

// CreateIdempotent вставляет сообщение с дедупом по (chat_id, sender_id,
// client_message_id). Без ключа (nil) — обычная вставка. С ключом: ON CONFLICT DO
// NOTHING; если строка не вставилась (конфликт), подгружаем исходное сообщение в
// obj, чтобы вернуть тот же результат, что и в первый раз (идемпотентный ответ).
func (r *MessageRepository) CreateIdempotent(ctx context.Context, obj *message.Message) (bool, error) {
	if obj.ClientMessageID == nil {
		return true, r.db.WithContext(ctx).Create(obj).Error
	}

	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "chat_id"}, {Name: "sender_id"}, {Name: "client_message_id"}},
			DoNothing: true,
		}).
		Create(obj)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		return true, nil
	}

	// Конфликт: строка уже была. Возвращаем исходное сообщение через тот же obj.
	var existing message.Message
	err := r.db.WithContext(ctx).
		Where("chat_id = ? AND sender_id = ? AND client_message_id = ?",
			obj.ChatID, obj.SenderID, *obj.ClientMessageID).
		First(&existing).Error
	if err != nil {
		return false, err
	}
	*obj = existing
	return false, nil
}

// Delete — soft-delete через gorm.DeletedAt: сообщение остаётся в БД
// («сообщение удалено»), но исключается из обычных выборок.
func (r *MessageRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Delete(&message.Message{}, id).Error
}

// Update сохраняет изменённые поля сообщения (Content, EditedAt).
func (r *MessageRepository) Update(ctx context.Context, obj *message.Message) error {
	return r.db.WithContext(ctx).
		Model(obj).
		Select("content", "edited_at").
		Updates(obj).Error
}

// GetMessages возвращает страницу истории чата через keyset-пагинацию по id DESC.
// Опирается на композитный индекс (chat_id, id). Возвращаются самые свежие сообщения
// первыми; для следующей страницы клиент передаёт LastMessageID последнего элемента.
func (r *MessageRepository) GetMessages(ctx context.Context, filter message.Filter) ([]*message.Message, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultMessagePageSize
	}

	q := r.db.WithContext(ctx).
		Where("chat_id = ?", filter.ChatID)

	if filter.LastMessageID != nil {
		q = q.Where("id < ?", *filter.LastMessageID)
	}

	var messages []*message.Message
	err := q.Order("id DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *MessageRepository) GetByID(ctx context.Context, id uint) (*message.Message, error) {
	var m message.Message
	err := r.db.WithContext(ctx).First(&m, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
