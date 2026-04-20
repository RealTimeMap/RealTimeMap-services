package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PgBlockedUserRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgBlockedUserRepository(db *gorm.DB, logger *zap.Logger) repository.BlockedUserRepository {
	return &PgBlockedUserRepository{
		db:     db,
		logger: logger,
	}
}
func (r *PgBlockedUserRepository) GetByID(ctx context.Context, userID uint, blockedUserID uint) (*model.BlockedUser, error) {
	var user *model.BlockedUser
	err := r.db.WithContext(ctx).Where("user_id = ? AND blocked_user_id = ?", userID, blockedUserID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.BlockedUserNotFound(blockedUserID)
		}
		return nil, err
	}

	return user, nil
}

func (r *PgBlockedUserRepository) Block(ctx context.Context, userID, blockedUserID uint) (bool, error) {
	payload := &model.BlockedUser{
		UserID:        userID,
		BlockedUserID: blockedUserID,
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(payload)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func (r *PgBlockedUserRepository) Unblock(ctx context.Context, data *model.BlockedUser) error {
	err := r.db.WithContext(ctx).Delete(&data).Error
	return err
}

func (r *PgBlockedUserRepository) GetBlockedUsers(ctx context.Context, userID uint, params pagination.Params) ([]uint, int64, error) {
	var users []uint
	var count int64
	query := r.db.WithContext(ctx).Model(&model.BlockedUser{}).
		Where("user_id = ?", userID)

	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(params.Offset()).Limit(params.Limit()).
		Pluck("blocked_user_id", &users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}
