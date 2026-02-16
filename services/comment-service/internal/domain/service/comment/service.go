package comment

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service"
	"go.uber.org/zap"
)

type Service struct {
	commentRepo repository.CommentRepository
	producer    service.EventPublisher

	logger *zap.Logger
}

func NewCommentService(commentRepo repository.CommentRepository, producer service.EventPublisher, logger *zap.Logger) *Service {
	return &Service{commentRepo: commentRepo, producer: producer, logger: logger}
}

// TODO В будущем добавить проверку существавание записи выбранной модели через gRPC

func (s *Service) Create(ctx context.Context, input CreateInput, userID uint) (*model.Comment, error) {
	if err := s.validateCreateInput(input); err != nil {
		return nil, err
	}

	comment := &model.Comment{
		UserID:     userID,
		Content:    input.Content,
		EntityType: model.EntityType(input.EntityType),
		EntityID:   input.EntityID,
		Depth:      0,
	}

	if input.ParentID != nil {
		parent, err := s.commentRepo.GetByID(ctx, *input.ParentID)
		if err != nil {
			return nil, err
		}

		if parent.Depth >= model.MaxDepth {
			return nil, domainerrors.CommentMaxDepthReached()
		}
		comment.ParentID = input.ParentID
		comment.Depth = parent.Depth + 1
	}

	newComment, err := s.commentRepo.Create(ctx, comment)
	if err != nil {
		return nil, err
	}

	go func() {
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.producer.PublishCommentCreated(publishCtx, newComment); err != nil {
			s.logger.Warn("Failed to publish comment created event", zap.Error(err))
		}
	}()
	return newComment, nil
}

func (s *Service) GetComments(ctx context.Context, filters model.CommentFilter) ([]*model.Comment, bool, error) {
	s.logger.Info("start PgCommentRepository.GetComments")

	comments, hasMore, err := s.commentRepo.GetComments(ctx, filters)
	if err != nil {
		return nil, false, err
	}

	return comments, hasMore, nil
}

func (s *Service) SoftDelete(ctx context.Context, userID, commentID uint) error {
	s.logger.Info("start PgCommentRepository.SoftDelete")

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

	comment.Content = model.OwnerDeletedContent
	comment.Status = model.CommentDeleted

	_, err = s.commentRepo.Update(ctx, comment)
	if err != nil {
		return err
	}
	// TODO kafka message
	return nil
}

func (s *Service) UpdateComment(ctx context.Context, input UpdateInput, userID, commentID uint) (*model.Comment, error) {
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

	comment.Content = input.Content
	newComment, err := s.commentRepo.Update(ctx, comment)
	if err != nil {
		return nil, err
	}
	return newComment, nil
}

func (s *Service) checkOwnerShip(userID uint, comment *model.Comment) error {
	if comment.UserID != userID {
		return domainerrors.NotCommentOwner()
	}
	return nil
}

func (s *Service) checkIsDeleted(comment *model.Comment) error {
	if comment.IsDeleted() {
		return domainerrors.CommentIsDeleted()
	}
	return nil
}

func (s *Service) validateCreateInput(input CreateInput) error {
	if len(input.Content) > 1024 {
		return domainerrors.ContentSooLong(len(input.Content))
	}
	return nil
}
