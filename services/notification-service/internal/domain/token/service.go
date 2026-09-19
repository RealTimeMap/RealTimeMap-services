package token

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

type Service struct {
	repo Repository

	logger *zap.Logger
}

func NewService(repo Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

type CreateTokenParams struct {
	UserID uint
	Token  string
}

func (s *Service) CreateOrNone(ctx context.Context, params CreateTokenParams) error {
	s.logger.Info("start CreateOrNone", zap.String("layer", "token service"), zap.Uint("userID", params.UserID))

	payload := &Model{
		UserID: params.UserID,
		Token:  params.Token,
	}

	err := s.repo.Create(ctx, payload)
	if err != nil {

		var conflictErr *apperror.ConflictError
		if errors.As(err, &conflictErr) {
			s.logger.Info("token already exist", zap.String("action", "skip"), zap.Uint("userID", params.UserID))
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) GetUserTokens(ctx context.Context, userID uint) ([]Model, error) {
	s.logger.Info("start GetUserTokens", zap.String("layer", "token service"), zap.Uint("userID", userID))

	objs, err := s.repo.GetByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(objs) < 1 {
		return nil, ErrNotFoundUserToken(userID)
	}
	return objs, nil
}
