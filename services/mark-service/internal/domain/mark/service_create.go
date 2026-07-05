package mark

import (
	"context"

	ctxHelper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

// Create Создание новой метки
func (s *Service) Create(ctx context.Context, user ctxHelper.UserInput, input CreateMarkParams) (*Mark, error) {
	// 1. Валидация входных данных
	if err := s.validateInput(ctx, user.UserID, input); err != nil {
		return nil, err
	}

	// 2. Загрузка фото в storage (если есть)
	var photos types.Photos
	if len(input.Photos) > 0 {
		uploadedPhotos, err := s.uploadPhotos(ctx, input.Photos)
		if err != nil {
			return nil, ErrStorageOperation("upload photos", err)
		}
		photos = uploadedPhotos
	}

	payload := &Mark{
		MarkName:       input.MarkName,
		AdditionalInfo: input.AdditionalInfo,
		StartAt:        input.StartAt,
		Geohash:        input.Geohash,
		Geom:           input.Geom,
		CategoryID:     input.CategoryId,
		Photos:         photos,
		UserID:         uint(user.UserID),
		UserName:       user.UserName,
	}
	if input.EndAt != nil {
		payload.EndAt = *input.EndAt
	} else {
		payload.DefaultEndAt()
	}

	// 3. Создание метки
	mark, err := s.markRepo.Create(ctx, payload)
	if err != nil {
		return nil, err
	}

	//// Асинхронная отправка события в Kafka (не блокируем ответ клиенту)
	//go s.shared.sendCreateEvent(context.Background(), mark_action) TODO добавить провайдер для KAFKA

	return mark, nil
}
