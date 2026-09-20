package mark

import (
	"context"

	"go.uber.org/zap"

	ctxHelper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

// Create Создание новой метки
func (s *Service) Create(ctx context.Context, user ctxHelper.UserInput, input CreateMarkParams) (*Mark, error) {
	// Метка без явного EndAt считается временной. Признак фиксируется до
	// валидации: validateDate проставляет дефолтный EndAt, после неё отличить
	// такую метку от метки с EndAt от клиента уже нельзя.
	isTemp := input.EndAt == nil

	// 1. Валидация входных данных + in feature: проверка расстояния метки и текущего положения пользователя
	if err := s.validateInput(ctx, user.UserID, &input); err != nil {
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

	// Страховка контракта: validateDate обязан проставить EndAt, если клиент его
	// не прислал. Пустой указатель здесь — баг валидации, а не данные клиента,
	// поэтому падать разыменованием в проде нельзя.
	if input.EndAt == nil {
		s.logger.Error("validateDate не проставил EndAt", zap.Time("startAt", input.StartAt))
		return nil, ErrEndAtNotResolved()
	}

	payload := &Mark{
		MarkName:       input.MarkName,
		AdditionalInfo: input.AdditionalInfo,
		StartAt:        input.StartAt,
		EndAt:          *input.EndAt, // validateDate гарантирует непустой EndAt
		Geohash:        input.Geohash,
		Geom:           input.Geom,
		CategoryID:     input.CategoryId,
		Photos:         photos,
		UserID:         uint(user.UserID),
		UserName:       user.UserName,
		IsTemp:         isTemp,
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
