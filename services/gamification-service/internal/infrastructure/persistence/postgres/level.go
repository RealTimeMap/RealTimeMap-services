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

type PgLevelRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPgLevelRepository(db *gorm.DB, logger *zap.Logger) repository.LevelRepository {
	return &PgLevelRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgLevelRepository) Create(ctx context.Context, level *model.Level) (*model.Level, error) {
	err := r.db.WithContext(ctx).Create(&level).Error
	if err != nil {
		return nil, err
	}
	return level, nil
}

func (r *PgLevelRepository) GetByLevel(ctx context.Context, level uint) (*model.Level, error) {
	var res *model.Level
	err := r.db.WithContext(ctx).First(&res, "level = ?", level).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.ErrLevelNotFount(level)
		}
		return nil, err
	}
	return res, nil
}

func (r *PgLevelRepository) GetAll(ctx context.Context) ([]*model.Level, error) {
	var res []*model.Level
	err := r.db.WithContext(ctx).Find(&res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}
