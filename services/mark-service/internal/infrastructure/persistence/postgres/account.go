package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/account"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/like"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
	"gorm.io/gorm"
)

type PgAccountRepository struct {
	db *gorm.DB
}

func NewPgAccountRepository(db *gorm.DB) account.Repository {
	return &PgAccountRepository{db: db}
}

func (r *PgAccountRepository) EraseUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Метки пользователя, включая мягко удалённые: подзапрос, а не список
		// id, чтобы не тащить в память все метки активного пользователя.
		userMarks := tx.Unscoped().Model(&mark.Mark{}).Select("id").Where("user_id = ?", userID)
		if err := tx.Where("user_id = ? OR mark_id IN (?)", userID, userMarks).
			Delete(&like.Reaction{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&mark.Mark{}).Error; err != nil {
			return err
		}

		// Связи личных меток со списками: join-таблица без модели, чистится
		// раньше обеих сторон.
		if err := tx.Exec(`
			DELETE FROM personal_marks_groups
			WHERE model_id IN (SELECT id FROM personal_marks WHERE user_id = @id)
			   OR group_id IN (SELECT id FROM groups WHERE user_id = @id)`,
			map[string]any{"id": userID}).Error; err != nil {
			return err
		}

		for _, m := range []any{&personal.Model{}, &personal.Group{}, &personal.Revision{}} {
			if err := tx.Unscoped().Where("user_id = ?", userID).Delete(m).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
