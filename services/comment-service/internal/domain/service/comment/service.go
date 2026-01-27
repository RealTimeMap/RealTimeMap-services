package comment

import (
	"context"
	"strconv"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/producer"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/repository"
	"go.uber.org/zap"
)

type Service struct {
	commentRepo repository.CommentRepository
	producer    *producer.Producer

	logger *zap.Logger
}

func NewCommentService(commentRepo repository.CommentRepository, producer *producer.Producer, logger *zap.Logger) *Service {
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

	s.sendCreateEvent(ctx, newComment)

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

func (s *Service) validateCreateInput(input CreateInput) error {
	if len(input.Content) > 1024 {
		return domainerrors.ContentSooLong(len(input.Content))
	}
	return nil
}

// sendCreateEvent отправляет событие создания комментария в Kafka.
func (s *Service) sendCreateEvent(ctx context.Context, comment *model.Comment) {
	if s.producer == nil {
		return
	}

	payload := events.NewCommentPayload(
		comment.ID,
		comment.UserID,
		comment.EntityID,
		string(comment.EntityType),
		comment.ParentID,
		comment.Content,
	)
	event := events.NewCommentCreated(payload)

	if err := s.producer.PublishWithMeta(ctx, producer.EventMeta{
		EventType: events.CommentCreated,
		UserID:    strconv.FormatUint(uint64(comment.UserID), 10),
		SourceID:  strconv.FormatUint(uint64(comment.ID), 10),
		Timestamp: time.Now().Format(time.RFC3339),
	}, event); err != nil {
		s.logger.Error("failed to publish comment.created event",
			zap.Uint("commentID", comment.ID),
			zap.Error(err),
		)
	}
}
