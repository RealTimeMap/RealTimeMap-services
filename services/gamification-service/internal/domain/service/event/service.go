package event

import (
	"context"
	"fmt"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/xp"
	"go.uber.org/zap"
)

type Service struct {
	eventRepo       repository.EventRuleRepository
	xpOperationRepo repository.XPOperationRepository
	userAchRepo     repository.UserAchievementCountRepository
	operator        *xp.XPOperator

	logger *zap.Logger
}

func NewEventGamificationService(eventRepo repository.EventRuleRepository, xpOperationRepo repository.XPOperationRepository, userAchRepo repository.UserAchievementCountRepository, operator *xp.XPOperator, logger *zap.Logger) *Service {
	return &Service{
		eventRepo:       eventRepo,
		xpOperationRepo: xpOperationRepo,
		userAchRepo:     userAchRepo,
		operator:        operator,
		logger:          logger,
	}
}

func (s *Service) GreatUserExp(ctx context.Context, userID uint, event string, sourceID *uint) error {
	rule, err := s.getRule(ctx, event)
	if err != nil {
		s.logger.Error("failed GetRule", zap.Error(err))
		return err
	}
	if err := s.checkDailyLimits(ctx, userID, rule); err != nil {
		s.logger.Info("Daily limits check failed", zap.Error(err))
		return err
	}
	input := xp.CreditInput{
		UserID:     userID,
		SourceType: model.SourceEvent,
		Amount:     int(rule.Reward.Amount),
		SourceID:   *sourceID,
	}
	result, err := s.operator.Credit(ctx, input)
	if err != nil {
		s.logger.Error("failed Credit", zap.Error(err))
		return err
	}

	err = s.userAchRepo.Increment(ctx, userID, event)
	if err != nil {
		s.logger.Error("failed Increment", zap.Error(err))
	}

	s.logger.Info(fmt.Sprintf("credit result: %v", result))
	return nil
}

func (s *Service) checkDailyLimits(ctx context.Context, userID uint, rule *model.EventRule) error {
	if rule == nil {
		return domainerrors.ErrRuleNotFount("checkDailyLimits")
	}
	if rule.IsRepeatable && rule.DailyLimit != nil {
		count, err := s.xpOperationRepo.GetCountEventsForDay(ctx, userID, time.Now())
		if err != nil {
			return err
		}
		if count >= int64(*rule.DailyLimit) {
			return domainerrors.ErrDailyLimitReached(*rule.DailyLimit)
		}
	}
	return nil
}

func (s *Service) getRule(ctx context.Context, event string) (*model.EventRule, error) {
	c, err := s.eventRepo.GetEventRuleByType(ctx, event)
	if err != nil {
		return nil, err
	}
	if !c.IsActive {
		return nil, domainerrors.ErrConfigNotActive(event)
	}
	return c, nil
}
