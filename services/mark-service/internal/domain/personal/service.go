package personal

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal/group"
	"go.uber.org/zap"
)

type Service struct {
	repo      Repository
	groupRepo group.Repository

	store  storage.Storage
	logger *zap.Logger
}

func NewService(repo Repository, groupRepo group.Repository, store storage.Storage, logger *zap.Logger) *Service {
	return &Service{
		repo:      repo,
		groupRepo: groupRepo,
		store:     store,
		logger:    logger,
	}
}

const maxPhotos int = 3

func (s *Service) uploadPhotos(ctx context.Context, photos []mediavalidator.PhotoInput) (types.Photos, error) {
	// Подготовка файлов для загрузки
	fileUploads := make([]storage.FileUpload, 0, len(photos))

	for _, photo := range photos {
		fileUploads = append(fileUploads, storage.FileUpload{
			Data: photo.Data,
			Options: storage.UploadOptions{
				FileName: photo.FileName,
				Category: storage.CategoryMarkPhoto,
				MaxSize:  5 * 1024 * 1024, // 5MB
				Optimize: false,           // Отключаем оптимизацию для ускорения
			},
		})
	}

	// Загрузка всех фото
	uploadedPhotos, err := s.store.UploadMultiple(ctx, fileUploads)
	if err != nil {
		return nil, err
	}

	return uploadedPhotos, nil
}
