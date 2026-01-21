package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgEventConfigRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPgEventConfigRepository(db *gorm.DB, logger *zap.Logger) repository.EventConfigRepository {
	return &PgEventConfigRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgEventConfigRepository) GetEventConfigByKafkaType(ctx context.Context, eventType string) (*model.EventConfig, error) {
	var eventConfig *model.EventConfig
	err := r.db.WithContext(ctx).First(&eventConfig, "kafka_event_type = ?", eventType).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrConfigNotFount(eventType)
		}
		return nil, err
	}
	return eventConfig, nil
}
