package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgReactionRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgReactionRepository(db *gorm.DB, logger *zap.Logger) repository.ReactionRepository {
	return &PgReactionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgReactionRepository) FindByUserAndComment(ctx context.Context, userID, commentID uint) (*model.Reaction, error) {
	r.logger.Info("start PgReactionRepository.FindByUserAndComment")

	var reaction model.Reaction
	err := DBFromCtx(ctx, r.db).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		First(&reaction).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &reaction, nil
}

func (r *PgReactionRepository) Create(ctx context.Context, reaction *model.Reaction) error {
	r.logger.Info("start PgReactionRepository.Create")
	return DBFromCtx(ctx, r.db).Create(reaction).Error
}

func (r *PgReactionRepository) Delete(ctx context.Context, id uint) error {
	r.logger.Info("start PgReactionRepository.Delete")
	return DBFromCtx(ctx, r.db).Unscoped().Delete(&model.Reaction{}, id).Error
}

func (r *PgReactionRepository) UpdateType(ctx context.Context, id uint, newType model.ReactionType) error {
	r.logger.Info("start PgReactionRepository.UpdateType")
	return DBFromCtx(ctx, r.db).Model(&model.Reaction{}).
		Where("id = ?", id).
		Update("type", newType).Error
}
