package comment

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/service"
	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/infrastructure/grpc/profile"
	"go.uber.org/zap"
)

type Service struct {
	commentRepo  repository.CommentRepository
	reactionRepo repository.ReactionRepository
	producer     service.EventPublisher
	txManager    service.TxManager

	profileAdapter *profile.Adapter
	logger         *zap.Logger
}

func NewCommentService(
	commentRepo repository.CommentRepository,
	reactionRepo repository.ReactionRepository,
	producer service.EventPublisher,
	txManager service.TxManager,
	profileAdapter *profile.Adapter,
	logger *zap.Logger,
) *Service {
	return &Service{
		commentRepo:    commentRepo,
		reactionRepo:   reactionRepo,
		producer:       producer,
		txManager:      txManager,
		profileAdapter: profileAdapter,
		logger:         logger,
	}
}

// TODO В будущем добавить проверку существавание записи выбранной модели через gRPC

func (s *Service) Create(ctx context.Context, input CreateInput, userID uint, username string) (*model.Comment, error) {
	if err := s.validateCreateInput(input); err != nil {
		return nil, err
	}

	comment := &model.Comment{
		UserID:     userID,
		Username:   username,
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

	s.attachAuthors(ctx, []*model.Comment{newComment})
	return newComment, nil
}

func (s *Service) GetComments(ctx context.Context, filters model.CommentFilter) ([]*model.Comment, bool, error) {
	s.logger.Info("start CommentService.GetComments")

	comments, hasMore, err := s.commentRepo.GetComments(ctx, filters)
	if err != nil {
		return nil, false, err
	}

	s.attachAuthors(ctx, comments)
	return comments, hasMore, nil
}

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

	s.attachAuthors(ctx, []*model.Comment{newComment})
	return newComment, nil
}

func (s *Service) ToggleReaction(ctx context.Context, input ToggleReactionInput, userID, commentID uint) (*model.ToggleResult, error) {
	s.logger.Info("start CommentService.ToggleReaction")

	comment, err := s.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment.IsDeleted() {
		return nil, domainerrors.CommentIsDeleted()
	}

	var result model.ToggleResult

	err = s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		existing, err := s.reactionRepo.FindByUserAndComment(txCtx, userID, commentID)
		if err != nil {
			return err
		}

		if existing == nil {
			// Реакции нет → создаём
			reaction := &model.Reaction{
				UserID:    userID,
				CommentID: commentID,
				Type:      input.Type,
			}
			if err := s.reactionRepo.Create(txCtx, reaction); err != nil {
				return err
			}
			if err := s.commentRepo.IncrementCounter(txCtx, commentID, counterColumn(input.Type), 1); err != nil {
				return err
			}
			result.Reaction = reaction
		} else if existing.Type == input.Type {
			// Та же реакция → удаляем
			if err := s.reactionRepo.Delete(txCtx, existing.ID); err != nil {
				return err
			}
			if err := s.commentRepo.IncrementCounter(txCtx, commentID, counterColumn(input.Type), -1); err != nil {
				return err
			}
			result.Reaction = nil
		} else {
			// Другая реакция → переключаем
			oldType := existing.Type
			if err := s.reactionRepo.UpdateType(txCtx, existing.ID, input.Type); err != nil {
				return err
			}
			if err := s.commentRepo.IncrementCounter(txCtx, commentID, counterColumn(oldType), -1); err != nil {
				return err
			}
			if err := s.commentRepo.IncrementCounter(txCtx, commentID, counterColumn(input.Type), 1); err != nil {
				return err
			}
			existing.Type = input.Type
			result.Reaction = existing
		}

		// Получаем актуальные счётчики
		updated, err := s.commentRepo.GetByID(txCtx, commentID)
		if err != nil {
			return err
		}
		result.LikesCount = updated.LikesCount
		result.DislikesCount = updated.DislikesCount

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &result, nil
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

func counterColumn(t model.ReactionType) string {
	if t == model.Like {
		return "likes_count"
	}
	return "dislikes_count"
}

func (s *Service) getUsersIDs(comments []*model.Comment) []uint {
	usersIDs := make([]uint, len(comments))
	for i, comment := range comments {
		usersIDs[i] = comment.UserID
	}
	return utils.UniqueValues(usersIDs)
}

func (s *Service) attachAuthors(ctx context.Context, comments []*model.Comment) {
	if len(comments) == 0 {
		return
	}

	ids := s.getUsersIDs(comments)
	profiles, err := s.profileAdapter.GetUserProfileByIDs(ctx, ids)

	byID := make(map[uint]*model.UserProfile, len(profiles))
	if err != nil {
		s.logger.Warn("profile-service degraded, using local author fallback", zap.Error(err))
	} else {
		for _, p := range profiles {
			byID[p.ID] = p
		}
	}

	for _, c := range comments {
		if p, ok := byID[c.UserID]; ok {
			c.Author = p
		} else {
			c.Author = localFallback(c)
		}
	}
}

func localFallback(c *model.Comment) *model.UserProfile {
	return &model.UserProfile{
		ID:       c.UserID,
		Username: c.Username,
	}
}
