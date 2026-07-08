package mark_interaction

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type ReactionCommand struct {
	MarkID uint
	UserID uint
}

// MarkLiker ставит реакцию на метку.
type MarkLiker interface {
	SetLike(ctx context.Context, params mark.AccrualParams) (mark.ReactionState, error)
}

// MarkUnliker снимает реакцию с метки.
type MarkUnliker interface {
	RemoveLike(ctx context.Context, params mark.AccrualParams) (mark.ReactionState, error)
}

type LikeMarkHandler struct {
	liker MarkLiker

	logger *zap.Logger
}

func NewLikeMarkHandler(liker MarkLiker, logger *zap.Logger) *LikeMarkHandler {
	return &LikeMarkHandler{
		liker:  liker,
		logger: logger,
	}
}

func (h *LikeMarkHandler) Handle(ctx context.Context, cmd ReactionCommand) (ReactionResult, error) {
	state, err := h.liker.SetLike(ctx, mark.AccrualParams{
		MarkID: cmd.MarkID,
		UserID: cmd.UserID,
	})
	if err != nil {
		return ReactionResult{}, err
	}
	return toReactionResult(state), nil
}

type UnlikeMarkHandler struct {
	unliker MarkUnliker

	logger *zap.Logger
}

func NewUnlikeMarkHandler(unliker MarkUnliker, logger *zap.Logger) *UnlikeMarkHandler {
	return &UnlikeMarkHandler{
		unliker: unliker,
		logger:  logger,
	}
}

func (h *UnlikeMarkHandler) Handle(ctx context.Context, cmd ReactionCommand) (ReactionResult, error) {
	state, err := h.unliker.RemoveLike(ctx, mark.AccrualParams{
		MarkID: cmd.MarkID,
		UserID: cmd.UserID,
	})
	if err != nil {
		return ReactionResult{}, err
	}
	return toReactionResult(state), nil
}
