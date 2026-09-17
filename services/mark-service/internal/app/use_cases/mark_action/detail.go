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
		// Мягкая деградация: метку отдаём с локальными данными владельца.
		//
		// Недоступность сервиса и отсутствие профиля — обе штатные ситуации.
		// Профиль заводится событием регистрации, и пока оно не доехало,
		// профиля нет; ронять на этом детальный просмотр метки нельзя —
		// сама метка при этом в базе есть и читается.
		if !errors.Is(err, profile.ErrUnavailable) && !errors.Is(err, profile.ErrNotFound) {
			return DetailMarkResult{}, err
		}
	}

	return toDetailMarkResult(res, p), nil
}

func (h *DetailMarkHandler) attachOwner(ctx context.Context, obj *mark.Mark) (UserResult, error) {
	p, err := h.provider.GetUserProfileByID(ctx, obj.UserID)

	if err != nil {
		// Отсутствие профиля — не авария: Warn, чтобы настоящие сбои gRPC не
		// утонули в шуме от меток пользователей без профиля.
		if errors.Is(err, profile.ErrNotFound) {
			h.logger.Warn("profile not found, falling back to local owner data",
				zap.Uint("user_id", obj.UserID))
		} else {
			h.logger.Error("gRPC failed to get user profile", zap.Error(err))
		}
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
		IsAdmin:  p.IsAdmin,
	}, nil

}

func localFallback(obj *mark.Mark) UserResult {
	return UserResult{
		ID:       obj.UserID,
		Username: obj.UserName,
	}
}
