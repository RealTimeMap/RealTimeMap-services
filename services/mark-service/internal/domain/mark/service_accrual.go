package mark

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/like"
	"go.uber.org/zap"
)

type AccrualService struct {
	markRepo        Repository
	interactionRepo like.Repository

	logger *zap.Logger
}

func NewAccrualService(markRepo Repository, interactionRepo like.Repository, logger *zap.Logger) *AccrualService {
	return &AccrualService{
		markRepo:        markRepo,
		interactionRepo: interactionRepo,
		logger:          logger,
	}
}

type AccrualParams struct {
	MarkID uint
	UserID uint
}

// ReactionState — актуальное состояние реакций метки для конкретного пользователя.
type ReactionState struct {
	Count   int64
	IsLiked bool
}

// MarkStat — агрегированная статистика метки.
// IsLiked отражает реакцию конкретного пользователя (false для анонима).
type MarkStat struct {
	Likes   int64
	Shares  int64
	IsLiked bool
}

func (s *AccrualService) IncreaseShare(ctx context.Context, params AccrualParams) (int64, error) {
	s.logger.Info("IncreaseShare", zap.Any("params", params))

	if err := checkMarkExist(ctx, s.markRepo.Exist, params.MarkID); err != nil {
		return 0, err
	}

	count, err := s.markRepo.IncShare(ctx, params.MarkID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// SetLike ставит реакцию пользователя на метку и возвращает актуальное состояние реакций.
// Возвращает like.AlreadyReacted, если реакция уже стоит.
func (s *AccrualService) SetLike(ctx context.Context, params AccrualParams) (ReactionState, error) {
	s.logger.Info("SetLike", zap.Any("params", params))

	if err := checkMarkExist(ctx, s.markRepo.Exist, params.MarkID); err != nil {
		return ReactionState{}, err
	}

	if err := s.interactionRepo.Like(ctx, params.MarkID, params.UserID); err != nil {
		return ReactionState{}, err
	}

	return s.reactionState(ctx, params)
}

// RemoveLike снимает реакцию пользователя с метки и возвращает актуальное состояние реакций.
// Идемпотентна.
func (s *AccrualService) RemoveLike(ctx context.Context, params AccrualParams) (ReactionState, error) {
	s.logger.Info("RemoveLike", zap.Any("params", params))

	if err := checkMarkExist(ctx, s.markRepo.Exist, params.MarkID); err != nil {
		return ReactionState{}, err
	}

	if err := s.interactionRepo.UnLike(ctx, params.MarkID, params.UserID); err != nil {
		return ReactionState{}, err
	}

	return s.reactionState(ctx, params)
}

// GetStat возвращает агрегированную статистику метки (лайки, шеры) и признак
// реакции запросившего пользователя. Для анонима (userID == 0) IsLiked всегда false.
func (s *AccrualService) GetStat(ctx context.Context, params AccrualParams) (MarkStat, error) {
	s.logger.Info("GetStat", zap.Any("params", params))

	obj, err := s.markRepo.GetByID(ctx, params.MarkID)
	if err != nil {
		return MarkStat{}, err
	}

	likes, err := s.interactionRepo.Count(ctx, params.MarkID)
	if err != nil {
		return MarkStat{}, err
	}

	var isLiked bool
	if params.UserID != 0 {
		isLiked, err = s.interactionRepo.IsLiked(ctx, params.MarkID, params.UserID)
		if err != nil {
			return MarkStat{}, err
		}
	}

	return MarkStat{
		Likes:   likes,
		Shares:  obj.SharedCount,
		IsLiked: isLiked,
	}, nil
}

// reactionState собирает актуальное число реакций и признак реакции пользователя.
func (s *AccrualService) reactionState(ctx context.Context, params AccrualParams) (ReactionState, error) {
	count, err := s.interactionRepo.Count(ctx, params.MarkID)
	if err != nil {
		return ReactionState{}, err
	}

	isLiked, err := s.interactionRepo.IsLiked(ctx, params.MarkID, params.UserID)
	if err != nil {
		return ReactionState{}, err
	}

	return ReactionState{Count: count, IsLiked: isLiked}, nil
}
