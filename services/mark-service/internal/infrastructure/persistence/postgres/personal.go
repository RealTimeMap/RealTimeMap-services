package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

func (r *PgPersonalMarkRepository) Create(ctx context.Context, obj *personal.Model) error {
	r.logger.Info("start Create", zap.String("layer", "postgres repo"))

	err := r.db.WithContext(ctx).Create(&obj).Error
	if err != nil {
		return err
	}
	return nil
}
