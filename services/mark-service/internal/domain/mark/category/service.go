package category

import (
	"context"
	"errors"
	"regexp"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
)

type Service struct {
	categoryRepo Repository
	store        storage.Storage
}

func NewService(categoryRepo Repository) *Service {
	return &Service{
		categoryRepo: categoryRepo,
	}
}

type CreateCategoryParams struct {
	CategoryName string
	Color        string
	FileName     string
	Icon         string
}

func (s *Service) Create(ctx context.Context, input CreateCategoryParams) (*Category, error) {
	if err := s.validateInput(input); err != nil {
		return nil, err
	}

	// 2. Проверка бизнес-правил (уникальность)
	if err := s.checkUniqueness(ctx, input.CategoryName); err != nil {
		return nil, err
	}

	// 3. Создание категории
	category := &Category{
		CategoryName: input.CategoryName,
		Color:        input.Color,
		Icon:         input.Icon,
		IsActive:     true,
	}

	created, err := s.categoryRepo.Create(ctx, category)
	if err != nil {
		return nil, ErrDatabaseQuery("create category", err)
	}

	return created, nil
}

func (s *Service) GetAll(ctx context.Context) ([]*Category, error) {
	objs, err := s.categoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	return objs, nil
}

// validateInput - валидация формата и содержимого данных
func (s *Service) validateInput(input CreateCategoryParams) error {
	// Валидация имени
	if input.CategoryName == "" {
		return ErrCategoryNameRequired()
	}
	if len(input.CategoryName) > 64 {
		return ErrCategoryNameTooLong(input.CategoryName)
	}

	// Валидация цвета
	if input.Color == "" {
		return ErrCategoryColorRequired()
	}
	if !s.isValidHexColor(input.Color) {
		return ErrCategoryColorInvalid(input.Color)
	}

	return nil
}

// checkUniqueness - проверка бизнес-правила уникальности
func (s *Service) checkUniqueness(ctx context.Context, name string) error {
	existing, err := s.categoryRepo.GetByName(ctx, name)
	if err != nil {
		// Проверяем ПО ТИПУ, а не по значению
		var notFoundErr *apperror.NotFoundError
		if !errors.As(err, &notFoundErr) {
			// Это НЕ NotFoundError - значит реальная ошибка БД
			return ErrDatabaseQuery("get category by name", err)
		}
	}
	if existing != nil {
		return ErrCategoryAlreadyExists(name)
	}
	return nil
}

// Вспомогательные методы

func (s *Service) isValidHexColor(color string) bool {
	hexColorRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	return hexColorRegex.MatchString(color)
}
