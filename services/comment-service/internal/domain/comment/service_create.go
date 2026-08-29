package comment

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/domainerrors"
)

type CreateParams struct {
	Content    string
	EntityType string
	EntityID   uint
	ParentID   *uint
}

// Create создаёт комментарий (или ответ на комментарий при наличии ParentID).
func (s *Service) Create(ctx context.Context, params CreateParams, userID uint, username string) (*Comment, error) {
	s.logger.Info("start CommentService.Create")

	if err := s.validateCreateParams(params); err != nil {
		return nil, err
	}

	comment := &Comment{
		UserID:     userID,
		Username:   username,
		Content:    params.Content,
		EntityType: EntityType(params.EntityType),
		EntityID:   params.EntityID,
		Depth:      0,
	}

	if params.ParentID != nil {
		parent, err := s.commentRepo.GetByID(ctx, *params.ParentID)
		if err != nil {
			return nil, err
		}

		if parent.Depth >= MaxDepth {
			return nil, domainerrors.CommentMaxDepthReached()
		}
		comment.ParentID = params.ParentID
		comment.Depth = parent.Depth + 1
	}

	return s.commentRepo.Create(ctx, comment)
}

func (s *Service) validateCreateParams(params CreateParams) error {
	if len(params.Content) > 1024 {
		return domainerrors.ContentSooLong(len(params.Content))
	}
	return nil
}
