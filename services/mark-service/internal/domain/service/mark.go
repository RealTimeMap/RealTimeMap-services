package service

import (
	"bytes"
	"context"
	"fmt"
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

type UserInput struct {
	UserName string
	UserID   int
}

type MarkInput struct {
	MarkName       valueobject.MarkName
	AdditionalInfo *string
	CategoryId     int
	StartAt        time.Time
	Duration       valueobject.Duration
	Geom           types.Point
	Geohash        string
	Photos         []mediavalidator.PhotoInput
	UserInput
}

type MarkUpdateInput struct {
	MarkID         int
	MarkName       *valueobject.MarkName
	AdditionalInfo *string
	CategoryId     *int
	Duration       *valueobject.Duration

	PhotosToDelete []string
	Photos         []mediavalidator.PhotoInput

	UserInput
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

// validateLimit проверка дневных лимитов
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

// GetMarksInArea получение меток в области карты
func (s *MarkService) GetMarksInArea(ctx context.Context, filter repository.Filter) ([]*model.Mark, error) {
	marks, err := s.markRepo.GetMarksInArea(ctx, filter)
	if err != nil {
		return nil, err
	}
	return marks, nil

}

// GetMarksInCluster получение сгруппированных меток по кластерам для отображения при большой области карты
func (s *MarkService) GetMarksInCluster(ctx context.Context, filter repository.Filter) ([]*model.Cluster, error) {
	clusters, err := s.markRepo.GetMarksInCluster(ctx, filter)
	if err != nil {
		return nil, err
	}
	return clusters, nil
}

// CheckExistsMark проверяет существования метки
func (s *MarkService) CheckExistsMark(ctx context.Context, id int) error {
	exists, err := s.markRepo.Exist(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domainerrors.ErrMarkNotFound(id)
	}
	return nil
}

func (s *MarkService) DeleteMark(ctx context.Context, id int, user UserInput) error {
	mark, err := s.markRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if mark.UserID != user.UserID {
		return domainerrors.ErrPermissionDenied()
	}

	if err := s.markRepo.Delete(ctx, id); err != nil {
		return err
	}
	return nil
}

func (s *MarkService) UpdateMark(ctx context.Context, input MarkUpdateInput) (*model.Mark, error) {
	// 1. Получение и проверка прав
	mark, err := s.markRepo.GetByID(ctx, input.MarkID)
	if err != nil {
		return nil, err
	}
	if err = s.checkOwnerShip(mark, input.UserID); err != nil {
		return nil, err
	}

	// 2. Обработка фотографий
	updatedPhotos, err := s.updatePhotos(ctx, mark.Photos, input.Photos, input.PhotosToDelete)
	if err != nil {
		return nil, err
	}

	// 3. Применение обновлений
	s.applyUpdates(mark, input)
	mark.Photos = updatedPhotos

	// 4. Сохранение в БД
	newMark, err := s.markRepo.Update(ctx, input.MarkID, mark)
	if err != nil {
		return nil, err
	}
	return newMark, nil
}

func (s *MarkService) applyUpdates(mark *model.Mark, input MarkUpdateInput) {
	if input.MarkName != nil {
		mark.MarkName = input.MarkName.String()
	}
	if input.AdditionalInfo != nil {
		mark.AdditionalInfo = input.AdditionalInfo
	}
	if input.Duration != nil {
		mark.Duration = input.Duration.Int()
	}
}

func (s *MarkService) checkOwnerShip(mark *model.Mark, userID int) error {
	fmt.Println(userID)
	if mark.UserID != userID {
		return domainerrors.ErrPermissionDenied()
	}
	return nil
}

// updatePhotos обрабатывает обновление фотографий:
// 1. Удаляет старые фото из storage и массива
// 2. Загружает новые фото в storage
// 3. Возвращает обновленный массив фотографий
func (s *MarkService) updatePhotos(ctx context.Context, currentPhotos types.Photos, newPhotos []mediavalidator.PhotoInput, photosToDelete []string) (types.Photos, error) {
	// 1. Создаем map для быстрого поиска удаляемых фото (по URL)
	deleteMap := make(map[string]bool, len(photosToDelete))
	for _, url := range photosToDelete {
		deleteMap[url] = true
	}

	// 2. Фильтруем старые фото и удаляем из storage
	var keptPhotos types.Photos
	for _, photo := range currentPhotos {
		if deleteMap[photo.URL] {
			// Удаляем из storage (игнорируем ошибки, так как файл может быть уже удален)
			_ = s.store.Delete(ctx, photo.StorageKey)
		} else {
			// Сохраняем фото, которое не удаляется
			keptPhotos = append(keptPhotos, photo)
		}
	}

	// 3. Загружаем новые фото в storage
	var uploadedPhotos types.Photos
	if len(newPhotos) > 0 {
		var err error
		uploadedPhotos, err = s.uploadPhotos(ctx, newPhotos)
		if err != nil {
			return nil, domainerrors.ErrStorageOperation("upload photos", err)
		}
	}

	// 4. Объединяем старые (не удаленные) + новые
	resultPhotos := append(keptPhotos, uploadedPhotos...)

	// 5. Валидация общего количества фото
	if len(resultPhotos) > maxPhotosPerMark {
		return nil, domainerrors.ErrTooManyPhotos(len(resultPhotos), maxPhotosPerMark)
	}

	return resultPhotos, nil
}

func (s *MarkService) DetailMark(ctx context.Context, id int) (*model.Mark, error) {
	mark, err := s.markRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mark, nil
}
