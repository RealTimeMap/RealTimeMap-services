package comment

import "context"

type UpdateParams struct {
	Content string
}

// UpdateComment обновляет содержимое комментария владельца.
func (s *Service) UpdateComment(ctx context.Context, params UpdateParams, userID, commentID uint) (*Comment, error) {
	s.logger.Info("start CommentService.UpdateComment")

	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if err := s.checkOwnerShip(userID, comment); err != nil {
		return nil, err
	}
	if err := s.checkIsDeleted(comment); err != nil {
		return nil, err
	}

	comment.Content = params.Content
	return s.commentRepo.Update(ctx, comment)
}
