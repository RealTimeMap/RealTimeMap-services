package generics

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (r *Base[T]) GetByID(ctx context.Context, id uint) (*T, error) {
	r.logger.Info("start GetByID", zap.String("layer", r.layer))

	var obj T
	err := r.dbCtx(ctx).Where("id = ?", id).First(&obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &obj, nil
}
