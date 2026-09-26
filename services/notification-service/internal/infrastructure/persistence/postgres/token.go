package postgres

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/token"
)

type PgUserTokenRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgUserTokenRepository(db *gorm.DB, logger *zap.Logger) token.Repository {
	return &PgUserTokenRepository{
		db:     db,
		logger: logger,
	}
}

// Register привязывает устройство к пользователю.
//
// Транзакция из двух шагов, а не один Save: FCM выдаёт один токен всем
// аккаунтам на устройстве, поэтому перед записью надо отцепить его от прежнего
// владельца — иначе уведомления предыдущего пользователя продолжат уходить
// тому, кто вошёл после него.
func (r *PgUserTokenRepository) Register(ctx context.Context, obj *token.Model) error {
	r.logger.Info("start Register", zap.String("layer", "token repository"))

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("token = ? AND user_id <> ?", obj.Token, obj.UserID).
			Delete(&token.Model{}).Error
		if err != nil {
			return err
		}

		// DoUpdates перечисляет колонки поимённо: settings в списке нет, и
		// повторная регистрация устройства не сбрасывает его настройки — ради
		// этого устройство и отделено от токена.
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"token", "platform", "updated_at", "last_used_at",
			}),
		}).Create(obj).Error
	})
}

func (r *PgUserTokenRepository) GetByUser(ctx context.Context, userID uint) ([]token.Model, error) {
	r.logger.Info("start GetByUser", zap.String("layer", "token repository"))
	var objs []token.Model

	err := r.db.WithContext(ctx).Model(&token.Model{}).Where("user_id = ?", userID).Find(&objs).Error
	if err != nil {
		return nil, err
	}

	return objs, nil
}

func (r *PgUserTokenRepository) GetByDevice(ctx context.Context, userID uint, deviceID string) (*token.Model, error) {
	r.logger.Info("start GetByDevice", zap.String("layer", "token repository"))
	var obj token.Model

	err := r.db.WithContext(ctx).Model(&token.Model{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		First(&obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, token.ErrNotFoundDevice(deviceID)
		}
		return nil, err
	}

	return &obj, nil
}

func (r *PgUserTokenRepository) UpdateSettings(ctx context.Context, id uint, settings token.Settings) error {
	r.logger.Info("start UpdateSettings", zap.String("layer", "token repository"))

	return r.db.WithContext(ctx).Model(&token.Model{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"settings":   settings,
			"updated_at": time.Now(),
		}).Error
}

func (r *PgUserTokenRepository) DeleteByUser(ctx context.Context, userID uint) (int64, error) {
	res := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&token.Model{})
	return res.RowsAffected, res.Error
}
