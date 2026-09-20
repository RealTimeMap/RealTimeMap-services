package mark

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
)

func (s *Service) validateInput(ctx context.Context, userID int, input *CreateMarkParams) error {
	// 1. Валидация категории (существует и активна)
	category, err := s.categoryRepo.GetByID(ctx, input.CategoryId)
	if err != nil {
		return err // ErrCategoryNotFound уже обрабатывается в репозитории
	}
	if !category.IsActive {
		return domainerrors.ErrCategoryNotActive(input.CategoryId)
	}

	// Валидация лимитов
	if err := s.validateLimit(ctx, userID); err != nil {
		return err
	}

	if err := s.validateDate(input, time.Now()); err != nil {
		return err
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

const (
	maxStartInPast   = 7 * 24 * time.Hour
	maxStartInFuture = 3 * 24 * time.Hour
	maxEndInFuture   = 7 * 24 * time.Hour
	defaultMarkTTL   = time.Hour
	minimumMarkTTL   = 30 * time.Minute
)

func (s *Service) validateDate(obj *CreateMarkParams, now time.Time) error {
	if obj.StartAt.Before(now.Add(-maxStartInPast)) {
		return ErrStartAtTooOld(int(maxStartInPast / (24 * time.Hour)))
	}
	if obj.StartAt.After(now.Add(maxStartInFuture)) {
		return ErrStartAtTooFuture(int(maxStartInFuture / (24 * time.Hour)))
	}

	// Точка, от которой метка реально начинает жить
	effectiveStart := obj.StartAt
	if effectiveStart.Before(now) {
		effectiveStart = now
	}

	// EndAt не задан — проставляем дефолт. Проверки ниже пропускаем
	if obj.EndAt == nil {
		endAt := effectiveStart.Add(defaultMarkTTL)
		obj.EndAt = &endAt
		return nil
	}

	if !obj.EndAt.After(obj.StartAt) {
		return ErrEndAtBeforeStart(*obj.EndAt)
	}
	if !obj.EndAt.After(now) {
		return ErrEndAtInPast(*obj.EndAt)
	}
	if obj.EndAt.Sub(effectiveStart) < minimumMarkTTL {
		return ErrMarkTTLTooShort(int(minimumMarkTTL / time.Minute))
	}
	if obj.EndAt.After(now.Add(maxEndInFuture)) {
		return ErrEndAtMaxInFuture(int(maxEndInFuture/(24*time.Hour)), *obj.EndAt)
	}

	return nil
}
