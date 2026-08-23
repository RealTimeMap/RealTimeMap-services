package mark

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"go.uber.org/zap"
)

func (s *Service) validateInput(ctx context.Context, userID int, input CreateMarkParams) error {
	// 1. Валидация категории (существует и активна)
	category, err := s.categoryRepo.GetByID(ctx, input.CategoryId)
	if err != nil {
		return err // ErrCategoryNotFound уже обрабатывается в репозитории
	}
	if !category.IsActive {
		return domainerrors.ErrCategoryNotActive(input.CategoryId)
	}

	// Валидация лимитов
	err = s.validateLimit(ctx, userID)
	if err != nil {
		return err
	}

	// 4. Валидация start_at (не слишком в прошлом/будущем)
	now := time.Now().UTC()
	pastLimit := now.AddDate(0, 0, -maxStartAtPastDays)
	futureLimit := now.AddDate(0, 0, maxStartAtFutureDays)
	if input.StartAt.Before(pastLimit) {
		return domainerrors.ErrStartAtTooOld(maxStartAtPastDays)
	}
	if input.StartAt.After(futureLimit) {
		return domainerrors.ErrStartAtTooFuture(maxStartAtFutureDays)
	}

	return nil
}

// validateLimit проверка дневных лимитов
func (s *Service) validateLimit(ctx context.Context, userID int) error {
	createdCount, err := s.markRepo.TodayCreated(ctx, uint(userID))
	if err != nil {
		return err
	}
	if createdCount > maxMarksPerDay {
		return domainerrors.ErrDailyMarkLimitExceeded(maxMarksPerDay)
	}
	return nil
}

// uploadPhotos загружает все фото в storage.
//
// Fail-fast: ошибка любого файла рушит создание метки целиком. Раньше метка
// создавалась с частичным набором фото и отвечала 200 OK. Валидация файлов
// отработала выше, поэтому ошибка здесь — сбой инфраструктуры, а не проблема
// конкретного файла.
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

func (s *Service) checkOwnerShip(obj *Mark, userID uint) error {
	s.logger.Debug("checkOwnerShip", zap.Any("mark", obj), zap.Uint("userID", userID))
	if obj.UserID != userID {
		return domainerrors.ErrPermissionDenied()
	}
	return nil
}
