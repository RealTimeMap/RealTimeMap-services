package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
)

type EventConfigRepository interface {
	GetEventConfigByKafkaType(ctx context.Context, eventType string) (*model.EventConfig, error)
}
