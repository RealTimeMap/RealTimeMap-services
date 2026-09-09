package postgres

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type PgPersonalMarkRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgPersonalMarkRepository(db *gorm.DB, logger *zap.Logger) personal.Repository {
	return &PgPersonalMarkRepository{
		db:     db,
		logger: logger,
	}
}

// dbCtx возвращает транзакцию из контекста (если сервис обернул вызов в
// txmanager.WithTx) либо собственный пул.
func (r *PgPersonalMarkRepository) dbCtx(ctx context.Context) *gorm.DB {
	return txmanager.DBFromCtx(ctx, r.db)
}

func (r *PgPersonalMarkRepository) Create(ctx context.Context, obj *personal.Model) error {
	r.logger.Info("start Create", zap.String("layer", "postgres repo"))

	return r.dbCtx(ctx).Create(obj).Error
}

func (r *PgPersonalMarkRepository) List(ctx context.Context, userID uint, since *uint, upTo uint, limit int) ([]personal.Model, error) {
	r.logger.Info("start List", zap.String("layer", "postgres repo"))

	q := r.db.WithContext(ctx).
		Unscoped().
		Where("user_id = ?", userID).
		Where("revision <= ?", upTo).
		Order("revision ASC").
		Limit(limit + 1)

	if since != nil {
		q = q.Where("revision > ?", *since)
	} else {
		q = q.Where("deleted_at IS NULL")
	}

	var objs []personal.Model

	if err := q.Preload("Groups").Find(&objs).Error; err != nil {
		return nil, err
	}

	return objs, nil
}
