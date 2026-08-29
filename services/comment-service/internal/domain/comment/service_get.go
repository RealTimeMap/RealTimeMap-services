package comment

import "context"

// GetComments возвращает комментарии по фильтру и флаг наличия следующей страницы.
func (s *Service) GetComments(ctx context.Context, filters CommentFilter) ([]*Comment, bool, error) {
	s.logger.Info("start CommentService.GetComments")

	comments, hasMore, err := s.commentRepo.GetComments(ctx, filters)
	if err != nil {
		return nil, false, err
	}
	return comments, hasMore, nil
}

// GetByID возвращает комментарий по идентификатору.
func (s *Service) GetByID(ctx context.Context, id uint) (*Comment, error) {
	s.logger.Info("start CommentService.GetByID")
	return s.commentRepo.GetByID(ctx, id)
}
