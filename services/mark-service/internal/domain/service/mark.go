package service

import (
	"bytes"
	"context"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"

	_ "golang.org/x/image/webp"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
)

const (
	maxPhotosPerMark     = 10 // Максимум 10 фото
	maxStartAtPastDays   = 1  // Не более 1 дня назад
	maxStartAtFutureDays = 30 // Не более 30 дней вперед
)

type PhotoInput struct {
	Data     []byte
	FileName string
}
type MarkInput struct {
	MarkName       string
	AdditionalInfo *string
	CategoryId     int
	StartAt        time.Time
	Duration       int
	Geom           types.Point
	Geohash        string
	Photos         []PhotoInput
}
type MarkService struct {
	markRepo     repository.MarkRepository
	categoryRepo repository.CategoryRepository
	store        storage.Storage
}

func NewMarkService(markRepo repository.MarkRepository, categoryRepo repository.CategoryRepository, store storage.Storage) *MarkService {
	return &MarkService{
		markRepo:     markRepo,
		categoryRepo: categoryRepo,
		store:        store,
	}
}

func (s *MarkService) CreateMark(ctx context.Context, input MarkInput) (*model.Mark, error) {
	// 1. Валидация входных данных
	if err := s.validateInput(ctx, input); err != nil {
		return nil, err
	}

	// 2. Загрузка фото в storage (если есть)
	var photos types.Photos
	if len(input.Photos) > 0 {
		uploadedPhotos, err := s.uploadPhotos(ctx, input.Photos)
		if err != nil {
			return nil, domainerrors.ErrStorageOperation("upload photos", err)
		}
		photos = uploadedPhotos
	}

	// 3. Создание метки
	result, err := s.markRepo.Create(ctx, &model.Mark{
		MarkName:       input.MarkName,
		AdditionalInfo: input.AdditionalInfo,
		StartAt:        input.StartAt,
		Duration:       input.Duration,
		Geohash:        input.Geohash,
		Geom:           input.Geom,
		CategoryID:     input.CategoryId,
		Photos:         photos,
		UserID:         1,       // TODO: Получать из контекста аутентификации
		UserName:       "user1", // TODO: Получать из контекста аутентификации
	})
	if err != nil {
		return nil, err
	}

	// TODO: Будущее - проверка лимита создания меток в день

	return result, nil
}

func (s *MarkService) validateInput(ctx context.Context, input MarkInput) error {
	// 1. Валидация имени метки
	if input.MarkName == "" {
		return domainerrors.ErrMarkNameRequired()
	}
	if len(input.MarkName) < 3 {
		return domainerrors.ErrMarkNameTooShort(input.MarkName)
	}
	if len(input.MarkName) > 100 {
		return domainerrors.ErrMarkNameTooLong(input.MarkName)
	}

	// 2. Валидация категории (существует и активна)
	category, err := s.categoryRepo.GetByID(ctx, input.CategoryId)
	if err != nil {
		return err // ErrCategoryNotFound уже обрабатывается в репозитории
	}
	if !category.IsActive {
		return domainerrors.ErrCategoryNotActive(input.CategoryId)
	}

	// 3. Валидация duration (только разрешенные значения)
	if !s.isValidDuration(input.Duration) {
		return domainerrors.ErrInvalidDuration(input.Duration)
	}

	// 4. Валидация start_at (не слишком в прошлом/будущем)
	now := time.Now()
	pastLimit := now.AddDate(0, 0, -maxStartAtPastDays)
	futureLimit := now.AddDate(0, 0, maxStartAtFutureDays)

	if input.StartAt.Before(pastLimit) {
		return domainerrors.ErrStartAtTooOld(maxStartAtPastDays)
	}
	if input.StartAt.After(futureLimit) {
		return domainerrors.ErrStartAtTooFuture(maxStartAtFutureDays)
	}

	// 5. Валидация фото (опционально, но если есть - проверяем)
	if len(input.Photos) > maxPhotosPerMark {
		return domainerrors.ErrTooManyPhotos(len(input.Photos), maxPhotosPerMark)
	}

	// Валидация каждого фото
	for i, photo := range input.Photos {
		if err := s.validatePhoto(i, photo); err != nil {
			return err
		}
	}

	return nil
}

// isValidDuration проверяет что duration входит в разрешённый список
func (s *MarkService) isValidDuration(duration int) bool {
	for _, allowed := range model.AllowedDuration {
		if duration == allowed {
			return true
		}
	}
	return false
}

// validatePhoto проверяет что фото валидно (MIME type и можно декодировать)
func (s *MarkService) validatePhoto(index int, photo PhotoInput) error {
	// Проверка MIME type из реальных байтов (не из HTTP заголовка!)
	mimeType := http.DetectContentType(photo.Data)

	// Разрешенные MIME types
	allowedMimeTypes := []string{
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	isAllowed := false
	for _, allowed := range allowedMimeTypes {
		if mimeType == allowed {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return domainerrors.ErrPhotoInvalidMimeType(index, mimeType)
	}

	// Проверка что изображение можно декодировать
	_, _, err := image.Decode(bytes.NewReader(photo.Data))
	if err != nil {
		return domainerrors.ErrPhotoInvalidImage(index)
	}

	return nil
}

// uploadPhotos загружает все фото в storage
func (s *MarkService) uploadPhotos(ctx context.Context, photos []PhotoInput) (types.Photos, error) {
	// Подготовка файлов для загрузки
	fileUploads := make([]storage.FileUpload, 0, len(photos))

	for _, photo := range photos {
		fileUploads = append(fileUploads, storage.FileUpload{
			Reader: bytes.NewReader(photo.Data),
			Options: storage.UploadOptions{
				FileName:      photo.FileName,
				Category:      storage.CategoryMarkPhoto,
				MaxSize:       5 * 1024 * 1024, // 5MB
				GenerateThumb: true,
				ThumbWidth:    300,
				ThumbHeight:   300,
				Optimize:      true,
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
