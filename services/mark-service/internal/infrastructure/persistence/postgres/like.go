package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/like"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LikeRepository struct {
	db    *gorm.DB
	log   *zap.Logger
	layer string
}

func NewPgLikeRepository(db *gorm.DB, logger *zap.Logger) like.Repository {
	return &LikeRepository{
		db:    db,
		log:   logger,
		layer: "like_repository",
	}
}

// Like ставит реакцию. Повторная реакция того же пользователя на ту же метку возвращает like.AlreadyReacted.
func (r *LikeRepository) Like(ctx context.Context, markID, userID uint) error {
	reaction := &like.Reaction{MarkID: markID, UserID: userID}

	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(reaction)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return like.AlreadyReacted(userID)
		}
		r.log.Error("like create err", zap.String("layer", r.layer), zap.Error(res.Error))
		return res.Error
	}

	if res.RowsAffected == 0 {
		return like.AlreadyReacted(userID)
	}

	return nil
}

// UnLike снимает реакцию. Идемпотентна: отсутствие реакции ошибкой не считается
func (r *LikeRepository) UnLike(ctx context.Context, markID, userID uint) error {
	err := r.db.WithContext(ctx).
		Where("mark_id = ? AND user_id = ?", markID, userID).
		Delete(&like.Reaction{}).Error
	if err != nil {
		r.log.Error("unlike delete err", zap.String("layer", r.layer), zap.Error(err))
		return err
	}
	return nil
}

// Count возвращает общее число реакций на метку
func (r *LikeRepository) Count(ctx context.Context, markID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&like.Reaction{}).
		Where("mark_id = ?", markID).
		Count(&count).Error
	if err != nil {
		r.log.Error("like count err", zap.String("layer", r.layer), zap.Error(err))
		return 0, err
	}
	return count, nil
}

// IsLiked сообщает, поставил ли пользователь реакцию на метку
func (r *LikeRepository) IsLiked(ctx context.Context, markID, userID uint) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Model(&like.Reaction{}).
		Select("count(*) > 0").
		Where("mark_id = ? AND user_id = ?", markID, userID).
		Find(&exists).Error
	if err != nil {
		r.log.Error("like exists err", zap.String("layer", r.layer), zap.Error(err))
		return false, err
	}
	return exists, nil
}
