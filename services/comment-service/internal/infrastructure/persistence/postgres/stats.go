package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/date"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgStatisticRepositoryRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgStatisticRepositoryRepository(db *gorm.DB, logger *zap.Logger) repository.StatisticRepository {
	return &PgStatisticRepositoryRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgStatisticRepositoryRepository) GetCountsByPeriod(ctx context.Context, userID uint, params date.Resolved) (int64, int64, error) {
	r.logger.Info("PgStatisticRepositoryRepository.GetCountsByPeriod start", zap.Uint("user_id", userID))

	currentStart, currentEnd := params.Current()
	prevStart, prevEnd := params.Previous()
	if currentStart == nil || currentEnd == nil || prevStart == nil || prevEnd == nil {
		return 0, 0, date.ErrInvalidPeriod
	}

	var row struct {
		CurrentCount int64
		PrevCount    int64
	}

	err := r.db.WithContext(ctx).Model(&model.Comment{}).
		Select(
			"COUNT(*) FILTER (WHERE created_at >= ? AND created_at < ?) AS current_count, "+
				"COUNT(*) FILTER (WHERE created_at >= ? AND created_at < ?) AS prev_count",
			currentStart, currentEnd, prevStart, prevEnd,
		).
		Where("user_id = ?", userID).
		Where("created_at >= ? AND created_at < ?", prevStart, currentEnd).
		Scan(&row).Error
	if err != nil {
		r.logger.Error("PgStatisticRepositoryRepository.GetCountsByPeriod error", zap.Error(err))
		return 0, 0, err
	}

	return row.CurrentCount, row.PrevCount, nil
}

func (r *PgStatisticRepositoryRepository) GetAllUsersComments(ctx context.Context, userID uint) (int64, error) {
	r.logger.Info("PgStatisticRepositoryRepository.GetAllUsersComments start", zap.Uint("user_id", userID))
	var count int64

	err := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("deleted_at IS NULL").
		Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		r.logger.Error("PgStatisticRepositoryRepository.GetAllUsersComments error", zap.Error(err))
		return 0, err
	}
	return count, nil
}
