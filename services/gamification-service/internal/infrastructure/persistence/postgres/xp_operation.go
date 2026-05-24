package postgres

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgXPOperationRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgXPOperation(db *gorm.DB, logger *zap.Logger) repository.XPOperationRepository {
	return &PgXPOperationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgXPOperationRepository) Create(ctx context.Context, XPOperation *model.XPOperation) (*model.XPOperation, error) {
	r.logger.Info("PgXPOperationRepository.Create", zap.Any("data", XPOperation))
	err := r.db.WithContext(ctx).Model(&model.XPOperation{}).Create(&XPOperation).Error
	return XPOperation, err
}

func (r *PgXPOperationRepository) GetCountForDay(ctx context.Context, userID uint, sourceType model.SourceType, date time.Time) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.XPOperation{}).
		Where("user_id = ? AND source_type = ? AND DATE(created_at) = DATE(?)", userID, sourceType, date).
		Count(&count).Error
	return count, err
}

func (r *PgXPOperationRepository) GetCountEventsForDay(ctx context.Context, userID uint, date time.Time) (int64, error) {
	return r.GetCountForDay(ctx, userID, model.SourceEvent, date)
}
