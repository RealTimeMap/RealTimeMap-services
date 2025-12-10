package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger/sl"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain"
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
		fmt.Println(err)
		r.log.Error("create_category err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}
	return data, nil
}

func (r *CategoryRepository) GetByName(ctx context.Context, name string) (*model.Category, error) {
	r.log.Info("get_category_by_name in: ", sl.String("layer", r.layer), sl.String("name", name))

	var category model.Category
	err := r.db.WithContext(ctx).Where("category_name = ?", name).First(&category).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCategoryNotFound(name)
		}
		r.log.Error("get_category_by_name err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}

	return &category, nil
}
