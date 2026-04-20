package blockeduser

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
)

type Service struct {
	repo repository.BlockedUserRepository

	profileRepo repository.ProfileRepository

	logger *zap.Logger
}

func NewService(repo repository.BlockedUserRepository, profileRepo repository.ProfileRepository, logger *zap.Logger) *Service {
	return &Service{
		repo:        repo,
		profileRepo: profileRepo,
		logger:      logger,
	}
}

// BlockUser функция сервисного слоя для блокировки пользователей
func (s *Service) BlockUser(ctx context.Context, userID, blockedUserID uint) error {
	if err := s.checkForBlockYourSelf(userID, blockedUserID); err != nil {
		return err
	}

	if err := s.checkProfileExists(ctx, blockedUserID); err != nil {
		return err
	}

	created, err := s.repo.Block(ctx, userID, blockedUserID)
	if err != nil {
		return err
	}
	if !created {
		return domainerrors.UserAlreadyBlocked(blockedUserID)
	}
	return nil
}

func (s *Service) UnBlockUser(ctx context.Context, userID, blockedUserID uint) error {
	blockedUser, err := s.repo.GetByID(ctx, userID, blockedUserID)
	if err != nil {
		return err
	}
	if blockedUser == nil {
		return domainerrors.BlockedUserNotFound(blockedUserID)
	}

	err = s.repo.Unblock(ctx, blockedUser)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) GetBlockedUsersProfile(ctx context.Context, userID uint, params *BlockedSearchParams) ([]*model.Profile, int64, error) {
	params.Pagination.Defaults()

	profilesIDs, count, err := s.repo.GetBlockedUsers(ctx, userID, params.Pagination)
	if err != nil {
		return nil, 0, err
	}

	profiles, err := s.profileRepo.GetProfilesByIDs(ctx, profilesIDs)
	if err != nil {
		return nil, 0, err
	}

	return profiles, count, nil
}

func (s *Service) checkForBlockYourSelf(userID, blockedUserID uint) error {
	if userID == blockedUserID {
		return domainerrors.CantBlockYourSelf(blockedUserID)
	}
	return nil
}

func (s *Service) checkProfileExists(ctx context.Context, userID uint) error {
	_, err := s.profileRepo.GetProfile(ctx, userID)
	return err
}
