package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/account"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment/reaction"
	"gorm.io/gorm"
)

type PgAccountRepository struct {
	db *gorm.DB
}

func NewPgAccountRepository(db *gorm.DB) account.Repository {
	return &PgAccountRepository{db: db}
}

func (r *PgAccountRepository) EraseUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&reaction.Reaction{}).Error; err != nil {
			return err
		}

		// Unscoped: мягко удалённые комментарии тоже хранят имя автора.
		// UpdateColumns — чтобы не трогать updated_at: обезличивание не
		// правка комментария, и «изменён» на нём выглядело бы ложью.
		return tx.Unscoped().
			Model(&comment.Comment{}).
			Where("user_id = ?", userID).
			UpdateColumns(map[string]any{
				"user_id":  account.DeletedAuthorID,
				"username": "",
			}).Error
	})
}
