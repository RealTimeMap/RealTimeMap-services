package subscription

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
)

type Service struct {
	repo        repository.SubscriptionRepository
	profileRepo repository.ProfileRepository
	blockedRepo repository.BlockedUserRepository

	logger *zap.Logger
}

func NewService(
	repo repository.SubscriptionRepository,
	profileRepo repository.ProfileRepository,
	blockedRepo repository.BlockedUserRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:        repo,
		profileRepo: profileRepo,
		blockedRepo: blockedRepo,
		logger:      logger,
	}
}

// Subscribe оформляет подписку subscriberID на targetID.
func (s *Service) Subscribe(ctx context.Context, subscriberID, targetID uint) error {
	if subscriberID == targetID {
		return domainerrors.CantSubscribeYourSelf(targetID)
	}

	if err := s.checkProfileExists(ctx, targetID); err != nil {
		return err
	}

	if err := s.checkNotBlocked(ctx, subscriberID, targetID); err != nil {
		return err
	}

	created, err := s.repo.Subscribe(ctx, subscriberID, targetID)
	if err != nil {
		return err
	}
	if !created {
		return domainerrors.AlreadySubscribed(targetID)
	}

	return nil
}

// Unsubscribe отменяет подписку subscriberID на targetID.
func (s *Service) Unsubscribe(ctx context.Context, subscriberID, targetID uint) error {
	exists, err := s.repo.Exists(ctx, subscriberID, targetID)
	if err != nil {
		return err
	}
	if !exists {
		return domainerrors.SubscriptionNotFound(targetID)
	}

	return s.repo.Unsubscribe(ctx, subscriberID, targetID)
}

// GetSubscriptionsProfile профили, на которые подписан пользователь.
func (s *Service) GetSubscriptionsProfile(ctx context.Context, userID uint, params *SubscriptionSearchParams) ([]*model.Profile, int64, error) {
	params.Pagination.Defaults()

	ids, count, err := s.repo.GetSubscriptions(ctx, userID, params.Pagination)
	if err != nil {
		return nil, 0, err
	}

	return s.loadProfiles(ctx, ids, count)
}

// GetSubscribersProfile профили подписчиков пользователя.
func (s *Service) GetSubscribersProfile(ctx context.Context, userID uint, params *SubscriptionSearchParams) ([]*model.Profile, int64, error) {
	params.Pagination.Defaults()

	ids, count, err := s.repo.GetSubscribers(ctx, userID, params.Pagination)
	if err != nil {
		return nil, 0, err
	}

	return s.loadProfiles(ctx, ids, count)
}

// IsSubscribed признак подписки текущего пользователя на targetID.
// Нужен фронту, чтобы отрисовать состояние кнопки на чужом профиле.
func (s *Service) IsSubscribed(ctx context.Context, subscriberID, targetID uint) (bool, error) {
	return s.repo.Exists(ctx, subscriberID, targetID)
}

func (s *Service) loadProfiles(ctx context.Context, ids []uint, count int64) ([]*model.Profile, int64, error) {
	if len(ids) == 0 {
		return []*model.Profile{}, count, nil
	}

	profiles, err := s.profileRepo.GetProfilesByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	return profiles, count, nil
}

func (s *Service) checkProfileExists(ctx context.Context, userID uint) error {
	_, err := s.profileRepo.GetProfile(ctx, userID)
	return err
}

func (s *Service) checkNotBlocked(ctx context.Context, userID, targetID uint) error {
	blocked, err := s.blockedRepo.ExistsBetween(ctx, userID, targetID)
	if err != nil {
		return err
	}
	if blocked {
		return domainerrors.SubscriptionBlocked()
	}
	return nil
}
