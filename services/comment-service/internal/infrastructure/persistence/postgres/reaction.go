package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment/reaction"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PgReactionRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgReactionRepository(db *gorm.DB, logger *zap.Logger) reaction.Repository {
	return &PgReactionRepository{
		db:     db,
		logger: logger,
	}
}

// Like ставит лайк. Повторный лайк того же пользователя на тот же комментарий возвращает reaction.AlreadyReacted.
func (r *PgReactionRepository) Like(ctx context.Context, commentID, userID uint) error {
	r.logger.Info("start PgReactionRepository.Like")

	react := &reaction.Reaction{CommentID: commentID, UserID: userID}

	res := DBFromCtx(ctx, r.db).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(react)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return reaction.AlreadyReacted(userID)
		}
		r.logger.Error("error PgReactionRepository.Like", zap.Error(res.Error))
		return res.Error
	}

	if res.RowsAffected == 0 {
		return reaction.AlreadyReacted(userID)
	}

	return nil
}

// UnLike снимает лайк. Идемпотентна: отсутствие лайка ошибкой не считается.
func (r *PgReactionRepository) UnLike(ctx context.Context, commentID, userID uint) error {
	r.logger.Info("start PgReactionRepository.UnLike")

	err := DBFromCtx(ctx, r.db).
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Delete(&reaction.Reaction{}).Error
	if err != nil {
		r.logger.Error("error PgReactionRepository.UnLike", zap.Error(err))
		return err
	}
	return nil
}

// Count возвращает общее число лайков комментария.
func (r *PgReactionRepository) Count(ctx context.Context, commentID uint) (int64, error) {
	r.logger.Info("start PgReactionRepository.Count")

	var count int64
	err := DBFromCtx(ctx, r.db).
		Model(&reaction.Reaction{}).
		Where("comment_id = ?", commentID).
		Count(&count).Error
	if err != nil {
		r.logger.Error("error PgReactionRepository.Count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// IsLiked сообщает, поставил ли пользователь лайк на комментарий.
func (r *PgReactionRepository) IsLiked(ctx context.Context, commentID, userID uint) (bool, error) {
	r.logger.Info("start PgReactionRepository.IsLiked")

	var exists bool
	err := DBFromCtx(ctx, r.db).
		Model(&reaction.Reaction{}).
		Select("count(*) > 0").
		Where("comment_id = ? AND user_id = ?", commentID, userID).
		Find(&exists).Error
	if err != nil {
		r.logger.Error("error PgReactionRepository.IsLiked", zap.Error(err))
		return false, err
	}
	return exists, nil
}

// LikedByUser возвращает множество лайкнутых пользователем комментариев из
// переданного списка — одним запросом на страницу вместо IsLiked в цикле.
func (r *PgReactionRepository) LikedByUser(ctx context.Context, commentIDs []uint, userID uint) (map[uint]bool, error) {
	r.logger.Info("start PgReactionRepository.LikedByUser")

	liked := make(map[uint]bool, len(commentIDs))
	if len(commentIDs) == 0 || userID == 0 {
		return liked, nil
	}

	var ids []uint
	err := DBFromCtx(ctx, r.db).
		Model(&reaction.Reaction{}).
		Where("user_id = ? AND comment_id IN ?", userID, commentIDs).
		Pluck("comment_id", &ids).Error
	if err != nil {
		r.logger.Error("error PgReactionRepository.LikedByUser", zap.Error(err))
		return nil, err
	}

	for _, id := range ids {
		liked[id] = true
	}
	return liked, nil
}
