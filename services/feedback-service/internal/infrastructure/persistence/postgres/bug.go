package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgBugRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgBugRepository(db *gorm.DB, logger *zap.Logger) bug.Repository {
	return &PgBugRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgBugRepository) Create(ctx context.Context, data *bug.Model) error {
	return r.db.WithContext(ctx).Create(&data).Error
}

func (r *PgBugRepository) GetByID(ctx context.Context, id uint) (*bug.Model, error) {
	var record *bug.Model

	err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (r *PgBugRepository) GetList(ctx context.Context, filter bug.Filter) ([]bug.Model, error) {
	var records []bug.Model
	q := r.db.WithContext(ctx).Model(&bug.Model{})

	if filter.Tag != nil && *filter.Tag != "" {
		q = q.Where("tag = ?", *filter.Tag)
	}

	// То же самое лучше сделать и для статуса на всякий случай
	if filter.Status != nil && *filter.Status != "" {
		r.logger.Debug("Status filter applied", zap.String("status", string(*filter.Status)))
		q = q.Where("status = ?", *filter.Status)
	}

	q = q.Limit(filter.Pagination.Limit()).
		Offset(filter.Pagination.Offset()).
		Order("created_at DESC")
	err := q.Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}
