package service

import (
	"bytes"
	"context"
	"strconv"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/service/input"
)

// markShared содержит общие функции для работы с метками, используемые как UserMarkService, так и AdminMarkService
type markShared struct {
	store    storage.Storage
	producer *producer.Producer
}

func newMarkShared(store storage.Storage, producer *producer.Producer) *markShared {
	return &markShared{
		store:    store,
		producer: producer,
	}
}

// uploadPhotos загружает все фото в storage
func (s *markShared) uploadPhotos(ctx context.Context, photos []mediavalidator.PhotoInput) (types.Photos, error) {
	// Подготовка файлов для загрузки
	fileUploads := make([]storage.FileUpload, 0, len(photos))

	for _, photo := range photos {
		fileUploads = append(fileUploads, storage.FileUpload{
			Reader: bytes.NewReader(photo.Data),
			Options: storage.UploadOptions{
				FileName:      photo.FileName,
				Category:      storage.CategoryMarkPhoto,
				MaxSize:       5 * 1024 * 1024, // 5MB
				GenerateThumb: false,
				Optimize:      false, // Отключаем оптимизацию для ускорения
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

// updatePhotos обрабатывает обновление фотографий:
// 1. Удаляет старые фото из storage и массива
// 2. Загружает новые фото в storage
// 3. Возвращает обновленный массив фотографий
func (s *markShared) updatePhotos(ctx context.Context, currentPhotos types.Photos, newPhotos []mediavalidator.PhotoInput, photosToDelete []string, maxPhotos int) (types.Photos, error) {
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

	// 5. Валидация общего количества фото (если maxPhotos > 0)
	if maxPhotos > 0 && len(resultPhotos) > maxPhotos {
		return nil, domainerrors.ErrTooManyPhotos(len(resultPhotos), maxPhotos)
	}

	return resultPhotos, nil
}

// sendCreateEvent отсылает ивент в kafka при создании метки
func (s *markShared) sendCreateEvent(ctx context.Context, mark *model.Mark) {
	// Пропускаем если Kafka выключен (producer == nil)
	if s.producer == nil {
		return
	}

	payload := events.NewMarkPayload(mark.ID, mark.CategoryID, mark.UserID, mark.MarkName, mark.AdditionalInfo)
	event := events.NewMarkCreate(payload)
	_ = s.producer.PublishWithMeta(ctx, producer.EventMeta{
		EventType: "mark.created",
		UserID:    strconv.Itoa(mark.UserID),
		SourceID:  strconv.Itoa(mark.ID),
		Timestamp: time.Now().Format(time.RFC3339)}, event)
}

// applyUpdates вспомогательная функция для обновления полей метки
func applyUpdates(mark *model.Mark, input input.MarkUpdateInput) {
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
