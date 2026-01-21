package levelservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelgenerator"
	"go.uber.org/zap"
)

type LevelService struct {
	levelRepo repository.LevelRepository
	strategy  levelgenerator.LevelGenerator

	logger *zap.Logger
}

func NewLevelService(levelRepo repository.LevelRepository, strategy levelgenerator.LevelGenerator, logger *zap.Logger) *LevelService {
	return &LevelService{levelRepo: levelRepo, strategy: strategy, logger: logger}
}

func (s *LevelService) GetOrCreate(ctx context.Context, level uint) (*model.Level, error) {
	existLevel, err := s.levelRepo.GetByLevel(ctx, level)
	if err == nil {
		return existLevel, nil
	}
	var notFoundErr *apperror.NotFoundError
	if !errors.As(err, &notFoundErr) {
		return nil, err
	}

	xpRequired := s.strategy.CalculateExpForLevel(level)

	createdLevel, err := s.levelRepo.Create(ctx, &model.Level{
		Level:      level,
		XPRequired: xpRequired,
		Title:      fmt.Sprintf("Level %d", level),
	})
	if err != nil {
		return nil, err
	}
	return createdLevel, nil
}

func (s *LevelService) GetNextLevel(ctx context.Context, currentLevel uint) (*model.Level, error) {
	return s.GetOrCreate(ctx, currentLevel+1)
}

func (s *LevelService) GetLevels(ctx context.Context) ([]*model.Level, error) {
	return s.levelRepo.GetAll(ctx)
}
