package comment

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/domainerrors"
)

// Like ставит лайк на комментарий. Идемпотентен: повторный лайк не меняет счётчик.
func (s *Service) Like(ctx context.Context, commentID, userID uint) (int64, bool, error) {
	s.logger.Info("start CommentService.Like")

	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return 0, false, err
	}
	if comment.IsDeleted() {
		return 0, false, domainerrors.CommentIsDeleted()
	}

	if err := s.reactionRepo.Like(ctx, commentID, userID); err != nil {
		return 0, false, err
	}

	count, err := s.reactionRepo.Count(ctx, commentID)
	if err != nil {
		return 0, false, err
	}
	return count, true, nil
}

// Unlike снимает лайк с комментария. Идемпотентен: отсутствие лайка ошибкой не считается.
func (s *Service) Unlike(ctx context.Context, commentID, userID uint) (int64, bool, error) {
	s.logger.Info("start CommentService.Unlike")

	if err := s.reactionRepo.UnLike(ctx, commentID, userID); err != nil {
		return 0, false, err
	}

	count, err := s.reactionRepo.Count(ctx, commentID)
	if err != nil {
		return 0, false, err
	}
	return count, false, nil
}
