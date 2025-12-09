package service

import (
	"bytes"
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
)

type CategoryService struct {
	categoryRepo repository.CategoryRepository
	store        storage.Storage
}

func NewCategoryService(categoryRepo repository.CategoryRepository, store storage.Storage) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
		store:        store,
	}
}

type CategoryCreateInput struct {
	CategoryName string
	Color        string
	FileName     string
	IconData     []byte
}

func (s *CategoryService) CreateCategory(ctx context.Context, data CategoryCreateInput) (*model.Category, error) {
	if err := s.validateInput(data); err != nil {
		return nil, err
	}
	icon, err := s.store.Upload(ctx, bytes.NewReader(data.IconData), storage.UploadOptions{
		FileName:      data.FileName,
		Category:      storage.CategoryCategories,
		MaxSize:       5 * 1024 * 1024,
		GenerateThumb: true,
		ThumbWidth:    150,
		ThumbHeight:   150,
		Optimize:      true,
	})
	if err != nil {
		return nil, err
	}

	validData := &model.Category{
		CategoryName: data.CategoryName,
		Color:        data.Color,
		Icon:         *icon,
		IsActive:     true,
	}

	category, err := s.categoryRepo.Create(ctx, validData)
	if err != nil {
		return nil, err

	}
	return category, nil

}

func (s *CategoryService) validateInput(data CategoryCreateInput) error {
	if data.CategoryName == "" {
		return errors.New("category name is required")
	}

	return nil

}
