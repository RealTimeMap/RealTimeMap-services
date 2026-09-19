package postgres

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"gorm.io/gorm"

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

func (r *PgUserTokenRepository) Create(ctx context.Context, obj *token.Model) error {
	r.logger.Info("start Create", zap.String("layer", "token repository"))

	err := r.db.WithContext(ctx).Model(&token.Model{}).Create(&obj).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return token.ErrAlreadyExist(obj.Token)
		}
		return err
	}
	return nil
}
