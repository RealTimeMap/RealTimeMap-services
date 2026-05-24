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

type PgAchievementRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgAchievementRepository(db *gorm.DB, logger *zap.Logger) repository.AchievementRepository {
	return &PgAchievementRepository{
		db: db,

		logger: logger,
	}
}

func (r *PgAchievementRepository) Create(ctx context.Context, achievement *model.Achievement) (*model.Achievement, error) {
	err := r.db.WithContext(ctx).Model(&model.Achievement{}).Create(achievement).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			r.logger.Warn("achievement already exists")
			return nil, domainerrors.AchievementAlreadyExistError(achievement.Code)
		}
		return nil, err
	}
	return achievement, nil
}
func (r *PgAchievementRepository) GetByID(ctx context.Context, id uint) (*model.Achievement, error) {
	var achievement *model.Achievement
	err := r.db.WithContext(ctx).
		Preload("Reward").
		Preload("Next").
		First(&achievement, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.AchievementNotFoundError("id", id)
		}
		return nil, err
	}
	return achievement, nil
}

func (r *PgAchievementRepository) GetByCode(ctx context.Context, code string) (*model.Achievement, error) {
	var achievement *model.Achievement
	err := r.db.WithContext(ctx).
		Preload("Reward").
		Preload("Next").
		Where("code = ?", code).
		First(&achievement).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.AchievementNotFoundError("code", code)
		}
		return nil, err
	}
	return achievement, nil
}

func (r *PgAchievementRepository) ListNearestByUser(ctx context.Context, userID uint, limit int) ([]repository.NearestAchievement, error) {
	type row struct {
		AchievementID uint
		Current       uint
		Threshold     uint
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("achievements").
		Select("achievements.id AS achievement_id, c.event_count AS current, achievements.threshold AS threshold").
		Joins(`JOIN user_achievement_counts c
              ON c.event_type = achievements.trigger_event_type
              AND c.user_id = ?`, userID).
		Where("achievements.is_active = ?", true).
		Where("achievements.deleted_at IS NULL").
		Where("c.event_count > 0").
		Where("c.event_count < achievements.threshold").
		Where(`NOT EXISTS (
            SELECT 1 FROM user_achievements ua
            WHERE ua.user_id = ? AND ua.achievement_id = achievements.id
        )`, userID).
		Where(`NOT EXISTS (
            SELECT 1 FROM achievements parent
            JOIN user_achievement_counts pc
              ON pc.event_type = parent.trigger_event_type
              AND pc.user_id = ?
            WHERE parent.next_id = achievements.id
              AND parent.is_active = true
              AND parent.deleted_at IS NULL
              AND pc.event_count > 0
              AND pc.event_count < parent.threshold
              AND NOT EXISTS (
                  SELECT 1 FROM user_achievements ua2
                  WHERE ua2.user_id = ? AND ua2.achievement_id = parent.id
              )
        )`, userID, userID).
		Order("c.event_count::float / NULLIF(achievements.threshold, 0) DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []repository.NearestAchievement{}, nil
	}

	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.AchievementID)
	}

	var achs []model.Achievement
	err = r.db.WithContext(ctx).
		Preload("Reward").
		Preload("Next").
		Where("id IN ?", ids).
		Find(&achs).Error
	if err != nil {
		return nil, err
	}

	byID := make(map[uint]model.Achievement, len(achs))
	for _, a := range achs {
		byID[a.ID] = a
	}

	result := make([]repository.NearestAchievement, 0, len(rows))
	for _, row := range rows {
		ach, ok := byID[row.AchievementID]
		if !ok {
			continue
		}
		result = append(result, repository.NearestAchievement{
			Achievement: ach,
			Current:     row.Current,
			Threshold:   row.Threshold,
		})
	}
	return result, nil
}

func (r *PgAchievementRepository) ListUnlockableByEvent(ctx context.Context, userID uint, eventType string) ([]model.Achievement, error) {
	var achievements []model.Achievement
	err := r.db.WithContext(ctx).
		Preload("Next").
		Preload("Reward").
		Joins(`LEFT JOIN user_achievement_counts c
              ON c.event_type = achievements.trigger_event_type
              AND c.user_id = ?`, userID).
		Joins(`LEFT JOIN user_achievements ua
              ON ua.achievement_id = achievements.id
              AND ua.user_id = ?`, userID).
		Where("achievements.trigger_event_type = ?", eventType).
		Where("achievements.is_active = ?", true).
		Where("COALESCE(c.event_count, 0) >= achievements.threshold").
		Where("ua.user_id IS NULL").
		Find(&achievements).Error
	if err != nil {
		return nil, err
	}
	return achievements, nil
}
