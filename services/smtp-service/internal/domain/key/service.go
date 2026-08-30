package key

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type ApiKeyService struct {
	repo Repository

	logger *zap.Logger
}

func NewApiKeyService(repo Repository, logger *zap.Logger) *ApiKeyService {
	return &ApiKeyService{
		repo:   repo,
		logger: logger,
	}
}

type CreateKeyParams struct {
	Name      string
	ExpiresAt *time.Time
}

func (s *ApiKeyService) ValidateToken(ctx context.Context, token string) error {
	obj, err := s.repo.Get(ctx, Hash(token))
	if err != nil {
		return err
	}
	now := time.Now()

	if obj.ExpiresAt != nil && obj.ExpiresAt.Before(now) {
		return ApiKeyExpired()
	}

	obj.LastUsedAt = &now
	err = s.repo.Update(ctx, obj)
	if err != nil {
		s.logger.Warn("Failed to update token used time", zap.Error(err))
	}

	return nil

}

func (s *ApiKeyService) CreateToken(ctx context.Context, data CreateKeyParams) (string, error) {
	token, hash, err := Generate()
	if err != nil {
		return "", err
	}
	obj := &Model{
		Name:      data.Name,
		Hint:      token[:14] + "...",
		KeyHash:   hash,
		ExpiresAt: data.ExpiresAt,
	}

	if err := s.repo.Create(ctx, obj); err != nil {
		return "", err
	}

	return token, nil
}
