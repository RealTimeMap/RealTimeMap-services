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

func (r *PgPersonalMarkRepository) List(ctx context.Context, userID uint)
