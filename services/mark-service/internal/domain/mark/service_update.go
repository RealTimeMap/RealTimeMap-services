package mark

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

func (s *Service) UpdateMark(ctx context.Context, input UpdateMarkParams, userID, markID uint) (*Mark, error) {
	// 1. Получение и проверка прав
	obj, err := s.markRepo.GetByID(ctx, markID)
	if err != nil {
		return nil, err
	}
	if err = s.checkOwnerShip(obj, userID); err != nil {
		return nil, err
	}

	// 2. Обработка фотографий (добавление новых + удаление старых)
	updatedPhotos, err := s.updatePhotos(ctx, obj.Photos, input.Photos, input.PhotosToDelete, maxPhotosPerMark)
	if err != nil {
		return nil, err
	}

	// 3. Применение обновлений
	s.applyUpdates(obj, input)
	obj.Photos = updatedPhotos

	// 4. Сохранение в БД
	newObj, err := s.markRepo.Update(ctx, markID, obj)
	if err != nil {
		return nil, err
	}
	return newObj, nil
}

// updatePhotos обрабатывает обновление фотографий:
// 1. Удаляет старые фото из storage и массива
// 2. Загружает новые фото в storage
// 3. Возвращает обновленный массив фотографий
func (s *Service) updatePhotos(ctx context.Context, currentPhotos types.Photos, newPhotos []mediavalidator.PhotoInput, photosToDelete []string, maxPhotos int) (types.Photos, error) {
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
			return nil, ErrStorageOperation("upload photos", err)
		}
	}

	// 4. Объединяем старые (не удаленные) + новые
	resultPhotos := append(keptPhotos, uploadedPhotos...)

	// 5. Валидация общего количества фото (если maxPhotos > 0)
	if maxPhotos > 0 && len(resultPhotos) > maxPhotos {
		return nil, ErrTooManyPhotos(len(resultPhotos), maxPhotos)
	}

	return resultPhotos, nil
}

func (s *Service) applyUpdates(mark *Mark, input UpdateMarkParams) {
	if input.MarkName != "" {
		mark.MarkName = input.MarkName
	}
	if input.AdditionalInfo != nil {
		mark.AdditionalInfo = input.AdditionalInfo
	}
}
