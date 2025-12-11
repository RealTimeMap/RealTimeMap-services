package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"regexp"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
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
} // TODO вынести

func (s *CategoryService) CreateCategory(ctx context.Context, input CategoryCreateInput) (*model.Category, error) {
	if err := s.validateInput(input); err != nil {
		return nil, err
	}

	// 2. Проверка бизнес-правил (уникальность)
	if err := s.checkUniqueness(ctx, input.CategoryName); err != nil {
		return nil, err
	}

	// 3. Загрузка иконки
	icon, err := s.uploadIcon(ctx, input.IconData, input.FileName)
	if err != nil {
		return nil, err
	}

	// 4. Создание категории
	category := &model.Category{
		CategoryName: input.CategoryName,
		Color:        input.Color,
		Icon:         *icon,
		IsActive:     true,
	}

	created, err := s.categoryRepo.Create(ctx, category)
	if err != nil {
		return nil, domainerrors.ErrDatabaseQuery("create category", err)
	}

	return created, nil
}

// validateInput - валидация формата и содержимого данных
func (s *CategoryService) validateInput(input CategoryCreateInput) error {
	// Валидация имени
	if input.CategoryName == "" {
		return domainerrors.ErrCategoryNameRequired()
	}
	if len(input.CategoryName) > 64 {
		return domainerrors.ErrCategoryNameTooLong(input.CategoryName)
	}

	// Валидация цвета
	if input.Color == "" {
		return domainerrors.ErrCategoryColorRequired()
	}
	if !s.isValidHexColor(input.Color) {
		return domainerrors.ErrCategoryColorInvalid(input.Color)
	}

	// Валидация файла
	if len(input.IconData) == 0 {
		return domainerrors.ErrCategoryIconRequired()
	}
	if err := s.validateImageContent(input.IconData); err != nil {
		return err
	}

	return nil
}

// checkUniqueness - проверка бизнес-правила уникальности
func (s *CategoryService) checkUniqueness(ctx context.Context, name string) error {
	existing, err := s.categoryRepo.GetByName(ctx, name)
	if err != nil {
		// Проверяем ПО ТИПУ, а не по значению
		var notFoundErr *apperror.NotFoundError
		if !errors.As(err, &notFoundErr) {
			// Это НЕ NotFoundError - значит реальная ошибка БД
			return domainerrors.ErrDatabaseQuery("get category by name", err)
		}
	}
	if existing != nil {
		return domainerrors.ErrCategoryAlreadyExists(name)
	}
	return nil
}

// uploadIcon - загрузка иконки в storage
func (s *CategoryService) uploadIcon(ctx context.Context, data []byte, filename string) (*types.Photo, error) {
	icon, err := s.store.Upload(ctx, bytes.NewReader(data), storage.UploadOptions{
		FileName:      filename,
		Category:      storage.CategoryCategories,
		MaxSize:       5 * 1024 * 1024,
		GenerateThumb: true,
		ThumbWidth:    150,
		ThumbHeight:   150,
		Optimize:      true,
	})
	if err != nil {
		if errors.Is(err, storage.ErrInvalidMimeType) {
			return nil, domainerrors.ErrCategoryIconMimeType("")
		}
		return nil, domainerrors.ErrStorageOperation("upload icon", err)
	}
	return icon, nil
}

// validateImageContent - валидация реального содержимого файла
func (s *CategoryService) validateImageContent(data []byte) error {
	// Проверка MIME типа из реальных байтов
	mimeType := http.DetectContentType(data)
	if !s.isAllowedMimeType(mimeType) {
		return domainerrors.ErrCategoryIconMimeType(mimeType)
	}

	// Проверка, что можно декодировать как изображение
	if mimeType != "image/svg+xml" {
		_, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return domainerrors.ErrCategoryIconInvalid()
		}
	}

	return nil
}

// Вспомогательные методы

func (s *CategoryService) isValidHexColor(color string) bool {
	hexColorRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	return hexColorRegex.MatchString(color)
}

func (s *CategoryService) isAllowedMimeType(mimeType string) bool {
	allowed := []string{"image/jpeg", "image/png", "image/webp", "image/svg+xml"}
	for _, t := range allowed {
		if t == mimeType {
			return true
		}
	}
	return false
}
