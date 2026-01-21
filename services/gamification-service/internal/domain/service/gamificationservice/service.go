package gamificationservice

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/levelservice"
	"go.uber.org/zap"
)

type GamificationService struct {
	levelService    *levelservice.LevelService
	eventRepo       repository.EventConfigRepository
	progressRepo    repository.UserProgressRepository
	userHistoryRepo repository.UserExpHistoryRepository

	logger *zap.Logger
}

func NewGamificationService(levelService *levelservice.LevelService, eventRepo repository.EventConfigRepository, progressRepo repository.UserProgressRepository, userHistoryRepo repository.UserExpHistoryRepository, logger *zap.Logger) *GamificationService {
	return &GamificationService{
		levelService:    levelService,
		eventRepo:       eventRepo,
		progressRepo:    progressRepo,
		userHistoryRepo: userHistoryRepo,
		logger:          logger,
	}
}

func (s *GamificationService) GreatUserExp(ctx context.Context, userID uint, event string, sourceID *uint) {
	s.logger.Info("start GamificationService.GreatUserExp", zap.String("event", event), zap.Uint("userID", userID))
	config, err := s.getConfig(ctx, event)
	if err != nil {
		s.logger.Error("exp not great", zap.Uint("userID", userID), zap.Error(err))
		return
	}
	uProgress, err := s.getOrCreateUserProgress(ctx, userID)
	if err != nil {
		return
	}
	// Проверка лимитов начисления для этого события
	if err := s.checkDailyLimits(ctx, userID, config, sourceID); err != nil {
		s.logger.Info("Daily limits check failed", zap.Error(err))
		return
	}

	updatedProgress, isLevelUp, err := s.updateProgress(ctx, uProgress, config.RewardExp)
	if err != nil {
		s.logger.Error("update progress failed", zap.Error(err))
		return
	}
	s.logger.Info("update progress successfully", zap.Bool("isLevelUp", isLevelUp), zap.Any("updatedProgress", updatedProgress))

	s.userHistoryRepo.Create(ctx, &model.UserExpHistory{UserID: userID, EarnedExp: config.RewardExp, Status: model.Credited, SourceID: sourceID, ConfigID: config.ID})
}

func (s *GamificationService) getConfig(ctx context.Context, event string) (*model.EventConfig, error) {
	s.logger.Info("start GamificationService.getConfig", zap.String("event", event))
	config, err := s.eventRepo.GetEventConfigByKafkaType(ctx, event)
	if err != nil {
		s.logger.Error("failed GamificationService.GreatUserExp", zap.Error(err))
		return nil, err
	}
	if !config.IsActive {
		s.logger.Warn("event is not active", zap.String("event", event))
		return nil, domainerrors.ErrConfigNotActive(event)
	}

	return config, nil
}

func (s *GamificationService) getOrCreateUserProgress(ctx context.Context, userID uint) (*model.UserProgress, error) {
	s.logger.Info("start GamificationService.getOrCreateUserProgress", zap.Uint("userID", userID))
	return s.progressRepo.GetOrCreate(ctx, userID)
}

func (s *GamificationService) checkDailyLimits(ctx context.Context, userID uint, config *model.EventConfig, sourceID *uint) error {
	if sourceID != nil && !config.IsRepeatable {
		exists, err := s.userHistoryRepo.ExistsBySourceID(ctx, userID, config.ID, *sourceID)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrAlreadyEarnedForSource()
		}
	}
	if config.DailyLimit != nil {
		count, err := s.userHistoryRepo.CountForDay(ctx, userID, config.ID, time.Now())
		if err != nil {
			s.logger.Error("failed checkDailyLimits", zap.Error(err))
			return err
		}
		if count >= int64(*config.DailyLimit) {
			return domainerrors.ErrDailyLimitReached(*config.DailyLimit)
		}
	}
	return nil
}

func (s *GamificationService) updateProgress(ctx context.Context, progress *model.UserProgress, xpReward uint) (updated *model.UserProgress, leveledUp bool, err error) {
	progress.CurrentXP += xpReward
	nextLevel, err := s.levelService.GetNextLevel(ctx, progress.CurrentLevel)
	if err != nil {
		return nil, false, err
	}
	if progress.CurrentXP >= nextLevel.XPRequired {
		progress.CurrentLevel = nextLevel.Level
		leveledUp = true
	}

	updated, err = s.progressRepo.Update(ctx, progress)
	if err != nil {
		return nil, false, err
	}

	return updated, leveledUp, nil
}
