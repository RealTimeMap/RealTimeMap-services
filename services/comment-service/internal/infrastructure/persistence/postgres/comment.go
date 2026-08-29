package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/domainerrors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgCommentRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgCommentRepository(db *gorm.DB, logger *zap.Logger) comment.Repository {
	return &PgCommentRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgCommentRepository) Create(ctx context.Context, c *comment.Comment) (*comment.Comment, error) {
	r.logger.Info("start PgCommentRepository.Create")
	err := DBFromCtx(ctx, r.db).Create(&c).Error
	if err != nil {
		r.logger.Error("error PgCommentRepository.Create", zap.Error(err), zap.Uint("id", c.ID))
		return nil, err
	}
	return c, nil
}

func (r *PgCommentRepository) GetByID(ctx context.Context, id uint) (*comment.Comment, error) {
	r.logger.Info("start PgCommentRepository.GetByID")
	var c *comment.Comment
	err := DBFromCtx(ctx, r.db).Preload("Parent").First(&c, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.CommentNotFound(id)
		}
		return nil, err
	}
	return c, nil
}

func (r *PgCommentRepository) GetComments(ctx context.Context, filters comment.CommentFilter) ([]*comment.Comment, bool, error) {
	r.logger.Info("start PgCommentRepository.GetComments")
	var comments []*comment.Comment

	query := DBFromCtx(ctx, r.db).
		Select("*, (SELECT COUNT(*) FROM comments r WHERE r.parent_id = comments.id AND r.deleted_at IS NULL AND r.status = ?) AS replies_count", comment.CommentActive).
		Where("entity_type = ? AND entity_id = ?", filters.Entity, filters.EntityID)

	if filters.ParentID != nil {
		query = query.Where("parent_id = ?", *filters.ParentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if filters.Cursor != nil {
		query = query.Where(filters.Sort.CursorCondition(), *filters.Cursor)
	}
	err := query.Order(filters.Sort.OrderClause()).Limit(filters.Limit + 1).Find(&comments).Error
	if err != nil {
		return nil, false, err
	}

	hasMore := len(comments) > filters.Limit
	if hasMore {
		comments = comments[:filters.Limit]
	}

	return comments, hasMore, nil
}

func (r *PgCommentRepository) Update(ctx context.Context, c *comment.Comment) (*comment.Comment, error) {
	r.logger.Info("start PgCommentRepository.Update")
	err := DBFromCtx(ctx, r.db).Save(&c).Error
	if err != nil {
		r.logger.Error("error PgCommentRepository.Update", zap.Error(err), zap.Uint("id", c.ID))
		return nil, err
	}
	return c, nil
}

func (r *PgCommentRepository) CountRelies(ctx context.Context, id uint) (int64, error) {
	r.logger.Info("start PgCommentRepository.CountRelies")
	var count int64
	err := DBFromCtx(ctx, r.db).Model(&comment.Comment{}).
		Where("parent_id = ? AND status = ?", id, comment.CommentActive).
		Count(&count).Error
	if err != nil {
		r.logger.Error("error PgCommentRepository.Count", zap.Error(err), zap.Uint("id", id))
		return 0, err
	}
	return count, nil
}
