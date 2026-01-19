package gamificationservice

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelgenerator"
	"go.uber.org/zap"
)

type GamificationService struct {
	levelRepo    repository.LevelRepository
	eventRepo    repository.EventConfigRepository
	progressRepo repository.UserProgressRepository
	expStrategy  levelgenerator.LevelGenerator

	logger *zap.Logger
}

func NewGamificationService(levelRepo repository.LevelRepository, eventRepo repository.EventConfigRepository, progressRepo repository.UserProgressRepository, strategy levelgenerator.LevelGenerator, logger *zap.Logger) *GamificationService {
	return &GamificationService{
		levelRepo:    levelRepo,
		eventRepo:    eventRepo,
		progressRepo: progressRepo,
		expStrategy:  strategy,
		logger:       logger,
	}
}

func (s *GamificationService) GreatUserExp(ctx context.Context, userID uint, event string) {
	s.logger.Info("start GamificationService.GreatUserExp", zap.String("event", event), zap.Uint("userID", userID))
	config, err := s.getConfig(ctx, event)
	if err != nil {
		s.logger.Warn("exp not great", zap.Uint("userID", userID), zap.Error(err))
		return
	}
	uProgress, err := s.getUserProgress(ctx, userID)
	if err != nil {
		return // TODO мб создавать профиль если нет? На сколько это будет правильно?
	}

	// Проверка лимитов начисления для этого события
	if err := s.checkDailyLimits(ctx, userID, config); err != nil {
		s.logger.Warn("reach daily limits", zap.Error(err))
		return
	}
	nLevel, err := s.levelRepo.GetByLevel(ctx, uProgress.CurrentLevel+1)
	if err != nil {
		s.logger.Warn("get nLevel", zap.Error(err))
		return
	}

	uProgress.CurrentXP += config.RewardExp

	if uProgress.CurrentXP >= nLevel.XPRequired {
		uProgress.CurrentLevel = nLevel.Level
	}

	newUProgress, err := s.progressRepo.Update(ctx, uProgress)
	if err != nil {
		s.logger.Warn("update nLevel", zap.Error(err))
		return
	}
	s.logger.Info("update nLevel", zap.Uint("userID", newUProgress.UserID))
}

func (s *GamificationService) getConfig(ctx context.Context, event string) (*model.EventConfig, error) {
	s.logger.Info("start GamificationService.getConfig", zap.String("event", event))
	config, err := s.eventRepo.GetEventConfigByKafkaType(ctx, event)
	if err != nil {
		s.logger.Error("failed GamificationService.GreatUserExp", zap.Error(err))
		return nil, err
	}
	if !config.IsActive {
		return nil, domainerrors.ErrConfigNotActive(event)
	}

	return config, nil
}

func (s *GamificationService) getUserProgress(ctx context.Context, userID uint) (*model.UserProgress, error) {
	s.logger.Info("start GamificationService.getUserProgress", zap.Uint("userID", userID))
	progress, err := s.progressRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return progress, nil
}

func (s *GamificationService) checkDailyLimits(ctx context.Context, userID uint, config *model.EventConfig) error {
	_ = time.Now()
	return nil
}
