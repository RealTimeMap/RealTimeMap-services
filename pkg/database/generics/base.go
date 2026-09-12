package generics

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
)

type Entity interface {
	TableName() string
}

type Base[T Entity] struct {
	db     *gorm.DB
	logger *zap.Logger
	layer  string
}

func NewBase[T Entity](db *gorm.DB, logger *zap.Logger, layer string) Base[T] {
	return Base[T]{db: db, logger: logger, layer: layer}
}

func (r *Base[T]) dbCtx(ctx context.Context) *gorm.DB {
	return txmanager.DBFromCtx(ctx, r.db)
}
