package mark_action

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark"
	"go.uber.org/zap"
)

type DetailMarkGetter interface {
	GetDetailMark(ctx context.Context, markID uint) (*mark.Mark, error)
}

type ProfileProvider interface {
	GetUserProfileByID(ctx context.Context, id uint) (*profile.UserProfile, error)
}

type DetailMarkHandler struct {
	getter DetailMarkGetter

	provider ProfileProvider
	logger   *zap.Logger
}

func NewDetailMarkHandler(getter DetailMarkGetter, provider ProfileProvider, logger *zap.Logger) *DetailMarkHandler {
	return &DetailMarkHandler{
		getter:   getter,
		provider: provider,
		logger:   logger,
	}
}

func (h *DetailMarkHandler) Handle(ctx context.Context, markID uint) (DetailMarkResult, error) {
	h.logger.Info("start markUseCases.DetailMarkHandler.Handle", zap.Uint("markID", markID))
	res, err := h.getter.GetDetailMark(ctx, markID)

	if err != nil {
		return DetailMarkResult{}, err
	}

	p, err := h.attachOwner(ctx, res)

	if err != nil {
		if !errors.Is(err, profile.ErrUnavailable) {
			// Если ошибка не связана с доступностью сервиса выбрасываем ее выше
			// иначе мягкая деградация
			return DetailMarkResult{}, err
		}
	}

	return toDetailMarkResult(res, p), nil
}

func (h *DetailMarkHandler) attachOwner(ctx context.Context, obj *mark.Mark) (UserResult, error) {
	p, err := h.provider.GetUserProfileByID(ctx, obj.UserID)

	if err != nil {
		h.logger.Error("gRPC failed to get user profile", zap.Error(err))
		return localFallback(obj), err
	}

	if p == nil {
		h.logger.Warn("gRPC get profile ended with nil user")
		return localFallback(obj), nil
	}

	return UserResult{
		ID:       p.ID,
		Username: p.Username,
		Avatar:   p.Avatar,
		Tag:      p.Tag,
	}, nil

}

func localFallback(obj *mark.Mark) UserResult {
	return UserResult{
		ID:       obj.UserID,
		Username: obj.UserName,
	}
}
