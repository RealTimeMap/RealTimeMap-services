package postgres

import (
	"context"
	"errors"

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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bug.ErrBugNotFound(id)
		}
		return nil, err
	}

	return record, nil
}

// GetByTaskID ищет баг по привязке к задаче.
//
// Отсутствие записи — обычный исход, а не сбой: у задачи может не быть
// бага. Поэтому наверх уходит nil, а не ошибка «не найдено».
func (r *PgBugRepository) GetByTaskID(ctx context.Context, taskID uint) (*bug.Model, error) {
	var record bug.Model

	err := r.db.WithContext(ctx).First(&record, "task_id = ?", taskID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &record, nil
}

// Update сохраняет поля, которые меняет привязка к задаче.
//
// Колонки перечислены явно: Save записал бы модель целиком и затёр бы
// поля, которых у нас на руках может не быть в актуальном виде.
// Отдельно важно, что task_id нужно уметь сбрасывать в NULL — Updates
// со структурой нулевое значение пропустил бы, поэтому здесь карта.
func (r *PgBugRepository) Update(ctx context.Context, data *bug.Model) error {
	result := r.db.WithContext(ctx).
		Model(&bug.Model{}).
		Where("id = ?", data.ID).
		Updates(map[string]any{
			"status":  data.Status,
			"task_id": data.TaskID,
		})

	if result.Error != nil {
		r.logger.Error("update bug failed", zap.Uint("id", data.ID), zap.Error(result.Error))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return bug.ErrBugNotFound(data.ID)
	}
	return nil
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

	// Набор статусов задаёт перечень открытых багов: одним равенством
	// «new или in work» не выразить.
	if len(filter.Statuses) > 0 {
		q = q.Where("status IN ?", filter.Statuses)
	}

	if filter.OnlyUnlinked {
		q = q.Where("task_id IS NULL")
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
