package service

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
)

type AdminMarkService struct {
	markRepo       repository.MarkRepository
	categoryRepo   repository.CategoryRepository
	mediaValidator *mediavalidator.PhotoValidator
	shared         *markShared
}

func NewAdminMarkService(markRepo repository.MarkRepository,
	categoryRepo repository.CategoryRepository,
	store storage.Storage,
	producer *producer.Producer,
	validator *mediavalidator.PhotoValidator) *AdminMarkService {
	return &AdminMarkService{
		markRepo:       markRepo,
		categoryRepo:   categoryRepo,
		mediaValidator: validator,
		shared:         newMarkShared(store, producer),
	}
}

// Основные методы

// GetAll получение всех записей
func (s *AdminMarkService) GetAll(ctx context.Context, params pagination.Params) ([]*model.Mark, int64, error) {
	marks, count, err := s.markRepo.GetAll(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	return marks, count, nil
}
