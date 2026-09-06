package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PgSubscriptionRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgSubscriptionRepository(db *gorm.DB, logger *zap.Logger) repository.SubscriptionRepository {
	return &PgSubscriptionRepository{
		db:     db,
		logger: logger,
	}
}

// dbCtx возвращает транзакцию из контекста (если сервис обернул вызов в
// txmanager.WithTx) либо собственный пул.
func (r *PgSubscriptionRepository) dbCtx(ctx context.Context) *gorm.DB {
	return txmanager.DBFromCtx(ctx, r.db)
}

// Subscribe создаёт подписку. Повторная подписка не считается ошибкой на уровне
// БД: OnConflict DoNothing, а факт «подписка уже была» отдаётся через false.
func (r *PgSubscriptionRepository) Subscribe(ctx context.Context, subscriberID, targetID uint) (bool, error) {
	payload := &model.Subscription{
		SubscriberID: subscriberID,
		TargetID:     targetID,
	}

	result := r.dbCtx(ctx).Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(payload)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func (r *PgSubscriptionRepository) Unsubscribe(ctx context.Context, subscriberID, targetID uint) error {
	return r.dbCtx(ctx).
		Where("subscriber_id = ? AND target_id = ?", subscriberID, targetID).
		Delete(&model.Subscription{}).Error
}

func (r *PgSubscriptionRepository) Exists(ctx context.Context, subscriberID, targetID uint) (bool, error) {
	var count int64
	err := r.dbCtx(ctx).Model(&model.Subscription{}).
		Where("subscriber_id = ? AND target_id = ?", subscriberID, targetID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetSubscriptions профили, на которые подписан userID.
func (r *PgSubscriptionRepository) GetSubscriptions(ctx context.Context, userID uint, params pagination.Params) ([]uint, int64, error) {
	return r.pluckPage(ctx, "subscriber_id = ?", userID, "target_id", params)
}

// GetSubscribers профили, подписанные на userID.
func (r *PgSubscriptionRepository) GetSubscribers(ctx context.Context, userID uint, params pagination.Params) ([]uint, int64, error) {
	return r.pluckPage(ctx, "target_id = ?", userID, "subscriber_id", params)
}

func (r *PgSubscriptionRepository) CountSubscribers(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.dbCtx(ctx).Model(&model.Subscription{}).
		Where("target_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *PgSubscriptionRepository) CountSubscriptions(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.dbCtx(ctx).Model(&model.Subscription{}).
		Where("subscriber_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *PgSubscriptionRepository) DeleteBetween(ctx context.Context, userID, otherID uint) error {
	return r.dbCtx(ctx).
		Where("(subscriber_id = ? AND target_id = ?) OR (subscriber_id = ? AND target_id = ?)",
			userID, otherID, otherID, userID).
		Delete(&model.Subscription{}).Error
}

// pluckPage возвращает страницу id из одной колонки с общим количеством записей.
// Порядок фиксируется по created_at, иначе постраничная выборка нестабильна.
func (r *PgSubscriptionRepository) pluckPage(
	ctx context.Context,
	where string,
	userID uint,
	column string,
	params pagination.Params,
) ([]uint, int64, error) {
	var (
		ids   []uint
		count int64
	)

	query := r.dbCtx(ctx).Model(&model.Subscription{}).Where(where, userID)

	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Offset(params.Offset()).
		Limit(params.Limit()).
		Pluck(column, &ids).Error
	if err != nil {
		return nil, 0, err
	}

	return ids, count, nil
}
