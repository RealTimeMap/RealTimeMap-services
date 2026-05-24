package txmanager

import (
	"context"

	"gorm.io/gorm"
)

// TxManager управляет транзакциями и позволяет репозиториям
// использовать общую транзакцию через контекст.
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) TxManager {
	return TxManager{db: db}
}

// WithTx оборачивает fn в транзакцию. Все репозитории, использующие
// DBFromCtx внутри fn, будут работать в рамках этой транзакции.
func (tm *TxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, ctxKey{}, tx)
		return fn(txCtx)
	})
}

// DBFromCtx извлекает *gorm.DB из контекста.
// Если транзакция есть — возвращает её, иначе — fallback.
func DBFromCtx(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(ctxKey{}).(*gorm.DB); ok {
		return tx
	}
	return fallback.WithContext(ctx)
}
