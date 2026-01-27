package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
)

type CommentRepository interface {
	// Create Создание комментария
	Create(ctx context.Context, comment *model.Comment) (*model.Comment, error)
	GetByID(ctx context.Context, id uint) (*model.Comment, error)
	GetComments(ctx context.Context, filters model.CommentFilter) ([]*model.Comment, bool, error)
}
