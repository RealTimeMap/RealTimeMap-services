package comment

import "context"

// SoftDelete помечает комментарий владельца как удалённый, заменяя содержимое.
func (s *Service) SoftDelete(ctx context.Context, userID, commentID uint) error {
	s.logger.Info("start CommentService.SoftDelete")

	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if err := s.checkOwnerShip(userID, comment); err != nil {
		return err
	}
	if err := s.checkIsDeleted(comment); err != nil {
		return err
	}

	comment.Content = OwnerDeletedContent
	comment.Status = CommentDeleted

	_, err = s.commentRepo.Update(ctx, comment)
	return err
}
