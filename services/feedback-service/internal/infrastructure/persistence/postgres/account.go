package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/account"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"gorm.io/gorm"
)

type PgAccountRepository struct {
	db *gorm.DB
}

func NewPgAccountRepository(db *gorm.DB) account.Repository {
	return &PgAccountRepository{db: db}
}

func (r *PgAccountRepository) EraseUser(ctx context.Context, userID uint) error {
	// Unscoped: мягко удалённые отчёты тоже хранят автора.
	// UpdateColumns — чтобы обезличивание не выглядело правкой отчёта.
	return r.db.WithContext(ctx).
		Unscoped().
		Model(&bug.Model{}).
		Where("user_id = ?", userID).
		UpdateColumns(map[string]any{
			"user_id":  nil,
			"ip":       "",
			"app_logs": "[]",
		}).Error
}
