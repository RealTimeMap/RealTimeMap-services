package achievement

import (
	"bytes"
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/service/xp"
	"go.uber.org/zap"
)

const NearestAchievementsLimit = 5

type Service interface {
	CreateAchievement(ctx context.Context, input Input) (*model.Achievement, error)
	OnEvent(ctx context.Context, userID uint, eventType string)
	GetAchievements(ctx context.Context, userID uint, params pagination.Params) ([]*model.UserAchievement, int64, error)
	GetNearestAchievements(ctx context.Context, userID uint) ([]repository.NearestAchievement, error)
}

type Input struct {
	Code         string
	Title        string
	Desc         string
	TriggerEvent string
	Threshold    uint
	Icon         mediavalidator.PhotoInput
	RewardID     uint
	NextID       *uint
}

type service struct {
	achievementRepo     repository.AchievementRepository
	rewardRepo          repository.XPRewardRepository
	userAchievementRepo repository.UserAchievementRepository

	xpOperator *xp.XPOperator

	store          storage.Storage
	photoValidator *mediavalidator.PhotoValidator

	logger *zap.Logger
}

func New(achievementRepo repository.AchievementRepository, rewardRepo repository.XPRewardRepository,
	userAchievementRepo repository.UserAchievementRepository, xpOperator *xp.XPOperator,
	store storage.Storage, photoValidator *mediavalidator.PhotoValidator,
	logger *zap.Logger) Service {
	return &service{
		achievementRepo:     achievementRepo,
		rewardRepo:          rewardRepo,
		userAchievementRepo: userAchievementRepo,

		xpOperator: xpOperator,

		store:          store,
		photoValidator: photoValidator,
		logger:         logger,
	}
}

func (s *service) OnEvent(ctx context.Context, userID uint, eventType string) {
	candidates, err := s.achievementRepo.ListUnlockableByEvent(ctx, userID, eventType)
	if err != nil {
		s.logger.Error("OnEvent", zap.Error(err))
	}

	for _, ach := range candidates {
		if err = s.unlock(ctx, userID, ach); err != nil {
			s.logger.Error("unlock failed", zap.Uint("user_id", userID), zap.Uint("achievement_id", ach.ID), zap.Error(err))
			continue
		}
	}
	s.logger.Debug("OnEvent", zap.Any("candidates", candidates))
}

func (s *service) unlock(ctx context.Context, userID uint, ach model.Achievement) error {
	s.logger.Info("unlock achievement", zap.Uint("user_id", userID), zap.Uint("achievement_id", ach.ID))
	if err := s.userAchievementRepo.Create(ctx, userID, ach.ID); err != nil {
		return err
	}

	if ach.Reward.Amount > 0 {
		_, err := s.xpOperator.Credit(ctx, xp.CreditInput{
			UserID:     userID,
			SourceType: model.SourceAchievement,
			SourceID:   ach.ID,
			Amount:     int(ach.Reward.Amount),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateAchievement создание ачивки
func (s *service) CreateAchievement(ctx context.Context, input Input) (*model.Achievement, error) {
	s.logger.Info("CreateAchievement", zap.String("code", input.Code))

	// Проверка конфига с наградой
	reward, err := s.rewardRepo.GetByID(ctx, input.RewardID)
	if err != nil {
		return nil, err
	}

	if err := s.photoValidator.ValidateSinglePhoto(input.Icon); err != nil {
		return nil, err
	}

	icon, err := s.store.Upload(ctx, bytes.NewReader(input.Icon.Data), storage.UploadOptions{
		FileName:      input.Icon.FileName,
		Category:      storage.CategoryAchievement,
		MaxSize:       5 * 1024 * 1024,
		Optimize:      true,
		GenerateThumb: false,
	})
	if err != nil {
		return nil, err
	}

	payload := &model.Achievement{
		Code:             input.Code,
		Title:            input.Title,
		Desc:             input.Desc,
		Threshold:        input.Threshold,
		TriggerEventType: input.TriggerEvent,
		IsActive:         true,
		Reward:           *reward,
		Icon:             *icon,
	}

	// Проверка следующего уровня если передан
	if input.NextID != nil {
		next, err := s.achievementRepo.GetByID(ctx, *input.NextID)
		if err != nil {
			return nil, err
		}
		payload.Next = next
	}

	achievement, err := s.achievementRepo.Create(ctx, payload)
	if err != nil {
		return nil, err
	}
	return achievement, nil
}

// GetAchievements получение ачивок с пагинацией
func (s *service) GetAchievements(ctx context.Context, userID uint, params pagination.Params) ([]*model.UserAchievement, int64, error) {
	params.Defaults()
	achs, count, err := s.userAchievementRepo.GetUserAchievements(ctx, userID, params)
	if err != nil {
		return nil, 0, err
	}

	return achs, count, nil
}

// GetNearestAchievements ближайшие к получению достижения пользователя
func (s *service) GetNearestAchievements(ctx context.Context, userID uint) ([]repository.NearestAchievement, error) {
	return s.achievementRepo.ListNearestByUser(ctx, userID, NearestAchievementsLimit)
}
