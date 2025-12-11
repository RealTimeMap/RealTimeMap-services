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
