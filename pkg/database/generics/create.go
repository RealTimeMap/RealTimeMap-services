package generics

import (
	"context"

	"go.uber.org/zap"
)

func (r *Base[T]) Create(ctx context.Context, obj *T) error {
	r.logger.Info("start Create", zap.String("layer", r.layer))
	return r.dbCtx(ctx).Create(obj).Error
}
