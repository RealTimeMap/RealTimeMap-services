package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"gorm.io/gorm"
)

type PgAccountRepository struct {
	db *gorm.DB
}

func NewPgAccountRepository(db *gorm.DB) repository.AccountRepository {
	return &PgAccountRepository{db: db}
}

func (r *PgAccountRepository) EraseUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unscoped: мягко удалённые начисления тоже привязаны к пользователю.
		for _, m := range []any{
			&model.XPOperation{},
			&model.UserAchievement{},
			&model.UserAchievementCount{},
			&model.UserProgress{},
		} {
			if err := tx.Unscoped().Where("user_id = ?", userID).Delete(m).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
