package comment

import "context"

// SoftDelete помечает комментарий владельца как удалённый, заменяя содержимое.
//
// Возвращает удалённый комментарий: вызывающему нужны его поля (автор,
// сущность, к которой он относится) для публикации события — перечитывать
// запись ради этого было бы лишним запросом.
func (s *Service) SoftDelete(ctx context.Context, userID, commentID uint) (*Comment, error) {
	s.logger.Info("start CommentService.SoftDelete")

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

	comment.Content = OwnerDeletedContent
	comment.Status = CommentDeleted

	return s.commentRepo.Update(ctx, comment)
}
