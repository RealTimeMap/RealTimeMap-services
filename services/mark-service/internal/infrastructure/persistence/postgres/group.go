package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal/group"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgGroupRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgGroupRepository(db *gorm.DB, logger *zap.Logger) group.Repository {
	return &PgGroupRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgGroupRepository) Create(ctx context.Context, obj *group.Model) error {
	r.logger.Info("start Create", zap.String("layer", "postgres repo"))
	err := r.db.WithContext(ctx).Create(obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return group.ErrAlreadyExistGroup(obj.Name)
		}
		return err
	}
	return nil
}

func (r *PgGroupRepository) Update(ctx context.Context, obj *group.Model) error {
	panic("implement me")
}

func (r *PgGroupRepository) List(ctx context.Context, userID uint, params pagination.Params) ([]*group.Model, int64, error) {
	r.logger.Info("start List", zap.String("layer", "postgres repo"))
	var count int64
	var objs []*group.Model
	err := r.db.WithContext(ctx).
		Model(&group.Model{}).
		Where("user_id = ?", userID).
		Offset(params.Offset()).
		Limit(params.Limit()).
		Find(&objs).
		Count(&count).
		Error
	if err != nil {
		return nil, 0, err
	}
	return objs, count, nil
}

func (r *PgGroupRepository) Delete(ctx context.Context, obj *group.Model) error {
	panic("implement me")
}

func (r *PgGroupRepository) GetBatch(ctx context.Context, userID uint, ids []uint) ([]*group.Model, error) {
	r.logger.Info("start GetBatch", zap.String("layer", "postgres repo"), zap.Any("ids", ids))
	var objs []*group.Model

	err := r.db.WithContext(ctx).Where("user_id = ? AND id IN ?", userID, ids).Find(&objs).Error
	if err != nil {
		return nil, err
	}

	if len(objs) != len(ids) {
		return nil, group.ErrNotFoundGroup(missing(ids, objs))
	}

	return objs, nil
}

func missing(ids []uint, rows []*group.Model) []uint {
	found := make(map[uint]struct{}, len(rows))
	for i := range rows {
		found[rows[i].ID] = struct{}{}
	}
	out := make([]uint, 0, len(ids)-len(rows))
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}
