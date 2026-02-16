package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgCommentRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgCommentRepository(db *gorm.DB, logger *zap.Logger) repository.CommentRepository {
	return &PgCommentRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgCommentRepository) Create(ctx context.Context, comment *model.Comment) (*model.Comment, error) {
	r.logger.Info("start PgCommentRepository.Create")
	err := r.db.WithContext(ctx).Create(&comment).Error
	if err != nil {
		r.logger.Error("error PgCommentRepository.Create", zap.Error(err), zap.Uint("id", comment.ID))
		return nil, err
	}
	return comment, nil
}

func (r *PgCommentRepository) GetByID(ctx context.Context, id uint) (*model.Comment, error) {
	r.logger.Info("start PgCommentRepository.GetByID")
	var comment *model.Comment
	err := r.db.WithContext(ctx).Preload("Parent").First(&comment, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainerrors.CommentNotFound(id)
		}
		return nil, err
	}
	return comment, nil
}

func (r *PgCommentRepository) GetComments(ctx context.Context, filters model.CommentFilter) ([]*model.Comment, bool, error) {
	r.logger.Info("start PgCommentRepository.GetComments")
	var comments []*model.Comment

	query := r.db.WithContext(ctx).
		Select("*, (SELECT COUNT(*) FROM comments r WHERE r.parent_id = comments.id AND r.deleted_at IS NULL AND r.status = ?) AS replies_count", model.CommentActive).
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

func (r *PgCommentRepository) Update(ctx context.Context, comment *model.Comment) (*model.Comment, error) {
	r.logger.Info("start PgCommentRepository.Update")
	err := r.db.WithContext(ctx).Save(&comment).Error
	if err != nil {
		r.logger.Error("error PgCommentRepository.Update", zap.Error(err), zap.Uint("id", comment.ID))
		return nil, err
	}
	return comment, nil
}

func (r *PgCommentRepository) CountRelies(ctx context.Context, id uint) (int64, error) {
	r.logger.Info("start PgCommentRepository.CountRelies")
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Comment{}).
		Where("parent_id = ? AND status = ?", id, model.CommentActive).
		Count(&count).Error
	if err != nil {
		r.logger.Error("error PgCommentRepository.Count", zap.Error(err), zap.Uint("id", id))
		return 0, err
	}
	return count, nil
}
