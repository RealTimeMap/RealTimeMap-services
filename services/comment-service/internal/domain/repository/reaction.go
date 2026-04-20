package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
)

type ReactionRepository interface {
	FindByUserAndComment(ctx context.Context, userID, commentID uint) (*model.Reaction, error)
	Create(ctx context.Context, reaction *model.Reaction) error
	Delete(ctx context.Context, id uint) error
	UpdateType(ctx context.Context, id uint, newType model.ReactionType) error
}
