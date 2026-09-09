package personal

import (
	"context"

	"go.uber.org/zap"
)

type RevisionService struct {
	repo RevisionRepository

	logger *zap.Logger
}

func NewRevisionService(repo RevisionRepository, logger *zap.Logger) *RevisionService {
	return &RevisionService{
		repo:   repo,
		logger: logger,
	}
}

func (s *RevisionService) ActualRevision(ctx context.Context, userID uint) (uint, error) {
	return s.repo.GetRevision(ctx, userID)
}
