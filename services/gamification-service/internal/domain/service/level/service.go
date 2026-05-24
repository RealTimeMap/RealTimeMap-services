package level

import (
	"context"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/level/generator"
	"go.uber.org/zap"
)

type Service struct {
	levelRepo repository.LevelRepository
	strategy  levelgenerator.LevelGenerator

	logger *zap.Logger
}

func NewLevelService(levelRepo repository.LevelRepository, strategy levelgenerator.LevelGenerator, logger *zap.Logger) *Service {
	return &Service{levelRepo: levelRepo, strategy: strategy, logger: logger}
}

// TODO сделать batch создание уровней

func (s *Service) GetOrCreate(ctx context.Context, level uint) (*model.Level, error) {
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

func (s *Service) GetNextLevel(ctx context.Context, currentLevel uint) (*model.Level, error) {
	return s.GetOrCreate(ctx, currentLevel+1)
}

func (s *Service) GetLevels(ctx context.Context) ([]*model.Level, error) {
	return s.levelRepo.GetAll(ctx)
}

func (s *Service) RecalculateLevel(ctx context.Context, progress *model.UserProgress) (bool, error) {
	s.logger.Info("Recalculating level", zap.Uint("user_id", progress.UserID), zap.Uint("current_level", progress.CurrentLevel))

	levelUp := false

	// Ищем следующий уровень на основе уровня пользователя
	nextLevel, err := s.GetNextLevel(ctx, progress.CurrentLevel)
	if err != nil {
		return false, err
	}

	// Проверям условия для повышения уровня, если что повышаем
	if progress.CurrentXP >= nextLevel.XPRequired {
		progress.CurrentLevel = nextLevel.Level
		levelUp = true
	}

	return levelUp, nil
}
