package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/message"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
