package progressservice

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"go.uber.org/zap"
)

type ProgressService struct {
	progressRepo repository.UserProgressRepository

	logger *zap.Logger
}

func NewProgressService(progressRepo repository.UserProgressRepository, logger *zap.Logger) *ProgressService {
	return &ProgressService{
		progressRepo: progressRepo,
		logger:       logger,
	}
}

func (s *ProgressService) GetLeaderBordUsers(ctx context.Context) ([]*model.UserProgress, error) {
	return s.progressRepo.GetTopUsers(ctx)
}
