package comment

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment/reaction"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/domainerrors"
	"go.uber.org/zap"
)

type Service struct {
	commentRepo  Repository
	reactionRepo reaction.Repository

	logger *zap.Logger
}

func NewService(commentRepo Repository, reactionRepo reaction.Repository, logger *zap.Logger) *Service {
	return &Service{
		commentRepo:  commentRepo,
		reactionRepo: reactionRepo,
		logger:       logger,
	}
}

func (s *Service) checkOwnerShip(userID uint, comment *Comment) error {
	if comment.UserID != userID {
		return domainerrors.NotCommentOwner()
	}
	return nil
}

func (s *Service) checkIsDeleted(comment *Comment) error {
	if comment.IsDeleted() {
		return domainerrors.CommentIsDeleted()
	}
	return nil
}
