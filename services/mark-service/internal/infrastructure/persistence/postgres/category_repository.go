package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger/sl"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db    *gorm.DB
	log   *zap.Logger
	layer string
}

func NewCategoryRepository(db *gorm.DB, log *zap.Logger) repository.CategoryRepository {
	return &CategoryRepository{db: db, log: log, layer: "category_repository"}
}

func (r *CategoryRepository) Create(ctx context.Context, data *model.Category) (*model.Category, error) {
	r.log.Info("create_category in: ", sl.String("layer", r.layer))
	err := r.db.WithContext(ctx).Create(&data).Error
	if err != nil {
		r.log.Error("create_category err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}
	return data, nil
}
