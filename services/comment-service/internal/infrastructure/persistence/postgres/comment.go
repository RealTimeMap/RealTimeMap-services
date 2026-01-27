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

	query := r.db.WithContext(ctx).Where("entity_type = ? AND entity_id = ? AND parent_id IS NULL", filters.Entity, filters.EntityID)

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
