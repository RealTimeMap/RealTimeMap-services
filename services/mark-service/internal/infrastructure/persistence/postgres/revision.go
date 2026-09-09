package postgres

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

type PgRevisionRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgRevisionRepository(db *gorm.DB, logger *zap.Logger) personal.RevisionRepository {
	return &PgRevisionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgRevisionRepository) dbCtx(ctx context.Context) *gorm.DB {
	return txmanager.DBFromCtx(ctx, r.db)
}

// UpsertRevision атомарно увеличивает счётчик ревизий пользователя и возвращает
// новое значение. Первый вызов создаёт строку с revision = 1.
func (r *PgRevisionRepository) UpsertRevision(ctx context.Context, userID uint) (uint, error) {
	r.logger.Info("start UpsertRevision", zap.String("layer", "postgres repo"))

	obj := personal.Revision{
		UserID:   userID,
		Revision: 1,
	}

	err := r.dbCtx(ctx).Clauses(
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{"revision": gorm.Expr(`"revisions"."revision" + 1`)}),
		},
		clause.Returning{Columns: []clause.Column{{Name: "revision"}}},
	).Create(&obj).Error
	if err != nil {
		return 0, err
	}

	return obj.Revision, nil
}

func (r *PgRevisionRepository) GetRevision(ctx context.Context, userID uint) (uint, error) {
	r.logger.Info("start GetRevision", zap.String("layer", "postgres repo"))
	var obj uint

	err := r.db.WithContext(ctx).
		Model(&personal.Revision{}).
		Where("user_id = ?", userID).
		Pluck("revision", &obj).
		Error
	if err != nil {
		return 0, err
	}
	return obj, nil
}
