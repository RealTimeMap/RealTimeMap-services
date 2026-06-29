package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PgAccrualRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgAccrualRepository(db *gorm.DB, logger *zap.Logger) repository.AccrualRepository {
	return &PgAccrualRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgAccrualRepository) IncShare(ctx context.Context, markID uint) (int64, error) {
	var mark model.Mark

	err := r.db.WithContext(ctx).
		Model(&mark).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "shared_count"}}}).
		Where("id = ?", markID).
		Update("shared_count", gorm.Expr("shared_count + 1")).Error
	if err != nil {
		return 0, err
	}

	return mark.SharedCount, nil
}

func (r *PgAccrualRepository) UnLike(ctx context.Context, markID, userID uint) error {
	//TODO implement me
	panic("implement me")
}

func (r *PgAccrualRepository) Like(ctx context.Context, markID, userID uint) error {
	payload := &model.MarkReaction{MarkID: markID, UserID: userID}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(payload).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domainerrors.ErrLikeAlreadySet()
		}
		return err
	}
	return nil
}
