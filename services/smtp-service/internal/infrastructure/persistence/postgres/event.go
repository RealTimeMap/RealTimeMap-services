package postgres

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgEmailEventRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgEmailEventRepository(db *gorm.DB, logger *zap.Logger) email.EventRepository {
	return &PgEmailEventRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgEmailEventRepository) Append(ctx context.Context, e *email.Event) error {
	if e.OccurredTime.IsZero() {
		e.OccurredTime = time.Now()
	}
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *PgEmailEventRepository) ListByEmailID(ctx context.Context, emailID uuid.UUID) ([]email.Event, error) {
	var events []email.Event

	err := r.db.WithContext(ctx).
		Where("email_id = ?", emailID).
		Order("occurred_time ASC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}
