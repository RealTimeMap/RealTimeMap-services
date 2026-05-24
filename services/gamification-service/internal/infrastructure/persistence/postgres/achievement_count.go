package postgres

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PgUserAchievementCountRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgUserAchievementCountRepository(db *gorm.DB, logger *zap.Logger) repository.UserAchievementCountRepository {
	return &PgUserAchievementCountRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgUserAchievementCountRepository) Create(ctx context.Context, progress *model.UserAchievementCount) (*model.UserAchievementCount, error) {
	err := r.db.WithContext(ctx).Model(&model.UserAchievementCount{}).Create(progress).Error
	if err != nil {
		return nil, err
	}
	return progress, nil
}

func (r *PgUserAchievementCountRepository) Increment(ctx context.Context, userID uint, eventType string) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "event_type"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"event_count": gorm.Expr("user_achievement_counts.event_count + 1"),
			"updated_at":  time.Now(),
		}),
	}).Create(&model.UserAchievementCount{
		UserID:    userID,
		EventType: eventType,
		Count:     1,
		UpdatedAt: time.Now(),
	}).Error

	return err
}
