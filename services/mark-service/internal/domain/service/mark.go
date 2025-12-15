package service

import (
	"bytes"
	"context"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	_ "golang.org/x/image/webp"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/valueobject"
)

const (
	maxPhotosPerMark     = 10    // Максимум 10 фото
	maxStartAtPastDays   = 1     // Не более 1 дня назад
	maxStartAtFutureDays = 30    // Не более 30 дней вперед
	maxMarksPerDay       = 10000 // Лимит на создание меток для пользователя
)

type MarkInput struct {
	MarkName       valueobject.MarkName
	AdditionalInfo *string
	CategoryId     int
	StartAt        time.Time
	Duration       valueobject.Duration
	Geom           types.Point
	Geohash        string
	Photos         []mediavalidator.PhotoInput
	UserName       string
	UserID         int
}
type MarkService struct {
	markRepo       repository.MarkRepository
	categoryRepo   repository.CategoryRepository
	store          storage.Storage
	producer       *producer.Producer
	mediaValidator *mediavalidator.PhotoValidator
}

func NewMarkService(markRepo repository.MarkRepository,
	categoryRepo repository.CategoryRepository,
	store storage.Storage,
	producer *producer.Producer,
	validator *mediavalidator.PhotoValidator) *MarkService {
	return &MarkService{
		markRepo:       markRepo,
		categoryRepo:   categoryRepo,
		store:          store,
		producer:       producer,
		mediaValidator: validator,
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
		MarkName:       input.MarkName.String(),
		AdditionalInfo: input.AdditionalInfo,
		StartAt:        input.StartAt,
		Duration:       input.Duration.Int(),
		Geohash:        input.Geohash,
		Geom:           input.Geom,
		CategoryID:     input.CategoryId,
		Photos:         photos,
		UserID:         input.UserID,
		UserName:       input.UserName,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *MarkService) validateInput(ctx context.Context, input MarkInput) error {
	// 1. Валидация категории (существует и активна)
	category, err := s.categoryRepo.GetByID(ctx, input.CategoryId)
	if err != nil {
		return err // ErrCategoryNotFound уже обрабатывается в репозитории
	}
	if !category.IsActive {
		return domainerrors.ErrCategoryNotActive(input.CategoryId)
	}

	// Валидация лимитов
	err = s.validateLimit(ctx, input.UserID)
	if err != nil {
		return err
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
	if err := s.mediaValidator.ValidatePhotos(input.Photos); err != nil {
		return err
	}

	return nil
}

// uploadPhotos загружает все фото в storage
func (s *MarkService) uploadPhotos(ctx context.Context, photos []mediavalidator.PhotoInput) (types.Photos, error) {
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

func (s *MarkService) validateLimit(ctx context.Context, userID int) error {
	createdCount, err := s.markRepo.TodayCreated(ctx, userID)
	if err != nil {
		return err
	}
	if createdCount > maxMarksPerDay {
		return domainerrors.ErrDailyMarkLimitExceeded(maxMarksPerDay)
	}
	return nil

}

func (s *MarkService) GetMarsInArea(ctx context.Context, filter repository.Filter) ([]*model.Mark, error) {
	marks, err := s.markRepo.GetMarksInArea(ctx, filter)
	if err != nil {
		return nil, err
	}
	return marks, nil

}
