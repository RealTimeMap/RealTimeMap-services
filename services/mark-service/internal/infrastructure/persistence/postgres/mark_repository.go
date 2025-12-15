package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger/sl"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MarkRepository struct {
	db    *gorm.DB
	log   *zap.Logger
	layer string
}

func NewMarkRepository(db *gorm.DB, logger *zap.Logger) repository.MarkRepository {
	return &MarkRepository{
		db:    db,
		log:   logger,
		layer: "mark_repository",
	}
}

func (r *MarkRepository) Create(ctx context.Context, data *model.Mark) (*model.Mark, error) {
	r.log.Info("create mark in: ", sl.String("layer", r.layer))

	// Создаем запись
	err := r.db.WithContext(ctx).Create(data).Error
	if err != nil {
		r.log.Error("create mark err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}

	// Загружаем связанную Category для возврата полного объекта
	err = r.db.WithContext(ctx).Preload("Category").First(data, data.ID).Error
	if err != nil {
		r.log.Error("failed to preload category: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}

	return data, nil
}

func (r *MarkRepository) TodayCreated(ctx context.Context, userID int) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.Mark{}).Where("user_id = ? AND DATE(created_at) = CURRENT_DATE", userID).Count(&count).Error
	if err != nil {
		r.log.Error("failed to get mark count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

func (r *MarkRepository) GetMarksInArea(ctx context.Context, filter repository.Filter) ([]*model.Mark, error) {
	var marks []*model.Mark

	err := r.db.WithContext(ctx).Model(&model.Mark{}).
		Preload("Category").
		Where("geohash IN (?)", filter.GeoHashes()).
		Where("start_at >= ?", filter.StartAt).
		Where("(start_at + interval '1 hour' * duration) >= ?", filter.EndAt).
		Find(&marks).Error
	if err != nil {
		r.log.Error("failed to get marks in area", zap.Error(err))
		return nil, err
	}

	return marks, nil
}
