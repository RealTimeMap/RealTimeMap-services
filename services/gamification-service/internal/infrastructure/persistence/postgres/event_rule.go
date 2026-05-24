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

type PgEventRuleRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPPgEventRuleRepository(db *gorm.DB, logger *zap.Logger) repository.EventRuleRepository {
	return &PgEventRuleRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgEventRuleRepository) GetEventRuleByType(ctx context.Context, eventType string) (*model.EventRule, error) {
	var eventRule *model.EventRule
	err := r.db.WithContext(ctx).Preload("Reward").First(&eventRule, "event_type = ?", eventType).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrRuleNotFount(eventType)
		}
	}
	return eventRule, nil
}
