package postgres

import (
	"context"
	"errors"

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

	q := r.dbCtx(ctx).
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

func (r *PgPersonalMarkRepository) GetByID(ctx context.Context, markID, userID uint) (personal.Model, error) {
	var obj personal.Model

	// First, а не Find: Find на пустой выборке не отдаёт ErrRecordNotFound и
	// вернул бы нулевую модель, которую вызывающий принял бы за чужую метку.
	err := r.dbCtx(ctx).
		Preload("Groups").
		Where("id = ?", markID).
		Where("user_id = ?", userID).
		First(&obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return personal.Model{}, personal.ErrNotFoundMark(markID)
		}
		return personal.Model{}, err
	}

	return obj, nil
}

func (r *PgPersonalMarkRepository) Update(ctx context.Context, obj *personal.Model) error {
	r.logger.Info("start Update", zap.String("layer", "postgres repo"))

	// Select по именам полей: Save/Updates со структурой пропустил бы false и
	// nil-поля (is_visible, description), а они здесь значимы.
	return r.dbCtx(ctx).
		Model(obj).
		Select(
			"geom",
			"title",
			"description",
			"category",
			"color",
			"icon",
			"is_visible",
			"photos",
			"revision",
		).
		Updates(obj).Error
}

// ReplaceGroups перезаписывает состав many2many-связи целиком.
func (r *PgPersonalMarkRepository) ReplaceGroups(ctx context.Context, obj *personal.Model, groups []*personal.Group) error {
	r.logger.Info("start ReplaceGroups", zap.String("layer", "postgres repo"))

	return r.dbCtx(ctx).
		Model(obj).
		Association("Groups").
		Replace(groups)
}

// SetRevision обновляет только revision. Unscoped, чтобы вызов после soft-delete
// тоже находил строку.
func (r *PgPersonalMarkRepository) SetRevision(ctx context.Context, markID, revision uint) error {
	r.logger.Info("start SetRevision", zap.String("layer", "postgres repo"))

	return r.dbCtx(ctx).
		Unscoped().
		Model(&personal.Model{}).
		Where("id = ?", markID).
		Update("revision", revision).Error
}

func (r *PgPersonalMarkRepository) Delete(ctx context.Context, markID uint) error {
	err := r.dbCtx(ctx).
		Delete(&personal.Model{}, markID).Error
	if err != nil {
		return err
	}

	return nil
}
