package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger/sl"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CategoryRepositoryV2 struct {
	db    *gorm.DB
	log   *zap.Logger
	layer string
}

func NewCategoryRepositoryV2(db *gorm.DB, log *zap.Logger) category.Repository {
	return &CategoryRepositoryV2{db: db, log: log, layer: "category_repository"}
}

func (r *CategoryRepositoryV2) Create(ctx context.Context, data *category.Category) (*category.Category, error) {
	r.log.Info("create_category in: ", sl.String("layer", r.layer))
	err := r.db.WithContext(ctx).Create(&data).Error
	if err != nil {
		r.log.Error("create_category err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}
	return data, nil
}

func (r *CategoryRepositoryV2) GetByName(ctx context.Context, name string) (*category.Category, error) {
	r.log.Info("get_category_by_name in: ", sl.String("layer", r.layer), sl.String("name", name))

	var category category.Category
	err := r.db.WithContext(ctx).Where("category_name = ?", name).First(&category).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrCategoryNotFound(name)
		}
		r.log.Error("get_category_by_name err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}

	return &category, nil
}

func (r *CategoryRepositoryV2) GetByID(ctx context.Context, id uint) (*category.Category, error) {
	r.log.Info("get_category_by_id in: ", sl.String("layer", r.layer), zap.Uint("id", id))

	var category category.Category
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrCategoryNotFound(id)
		}
		r.log.Error("get_category_by_id err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}

	return &category, nil
}

func (r *CategoryRepositoryV2) Exist(ctx context.Context, id int) (bool, error) {
	r.log.Info("check_exist_category_by_id", sl.String("layer", r.layer))
	var exists bool
	err := r.db.WithContext(ctx).
		Model(&category.Category{}).
		Select("1").
		Where("id = ? AND is_active = ?", id, true).
		Limit(1).
		Find(&exists).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, domainerrors.ErrCategoryNotFound(id)
		}
		r.log.Error("check_exist_category_by_id err: ", sl.String("layer", r.layer), zap.Error(err))
		return false, err
	}
	return exists, nil
}

func (r *CategoryRepositoryV2) GetAll(ctx context.Context) ([]*category.Category, error) {
	r.log.Info("get_category_by_name in: ", sl.String("layer", r.layer))
	var categories []*category.Category
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&categories).Error
	if err != nil {
		r.log.Error("get_category_by_name err: ", sl.String("layer", r.layer), zap.Error(err))
		return nil, err
	}
	return categories, nil
}
