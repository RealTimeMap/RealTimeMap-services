package profile

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
)

type Service struct {
	profileRepo repository.ProfileRepository

	logger *zap.Logger
}

func NewProfileService(profileRepo repository.ProfileRepository, logger *zap.Logger) *Service {
	return &Service{
		profileRepo: profileRepo,
		logger:      logger,
	}
}

func (s *Service) CreateProfile(ctx context.Context, input CreateProfileInput) (*model.Profile, error) {
	s.logger.Info("ProfileService.CreateProfile", zap.Uint("user_id", input.UserID))
	if err := s.checkProfileExists(ctx, input.UserID); err != nil {
		var notFoundErr *apperror.NotFoundError
		if !errors.As(err, &notFoundErr) {
			return nil, err
		}
	}

	payload := &model.Profile{
		UserID:          input.UserID,
		Username:        input.Username,
		IsPrivate:       false,
		PrivacySettings: model.DefaultPrivacySettings(),
	}

	profile, err := s.profileRepo.Create(ctx, payload)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *Service) checkProfileExists(ctx context.Context, userId uint) error {
	profile, err := s.profileRepo.GetProfile(ctx, userId)
	if err != nil {
		if errors.Is(err, domainerrors.ProfileNotFound(userId)) {
			return nil
		}

		return err
	}
	if profile != nil {
		return domainerrors.ProfileAlreadyExists(userId)
	}
	return nil
}

func (s *Service) GetProfile(ctx context.Context, userId uint) (*model.Profile, error) {
	s.logger.Info("ProfileService.GetProfile", zap.Uint("user_id", userId))
	profile, err := s.profileRepo.GetProfile(ctx, userId)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *Service) SearchProfiles(ctx context.Context, input *SearchProfilesInput) ([]*model.Profile, int64, error) {
	s.logger.Info("ProfileService.SearchProfiles", zap.String("username", input.Username))

	input.Pagination.Defaults()

	profiles, total, err := s.profileRepo.GetProfiles(ctx, input.Username, input.Pagination)
	if err != nil {
		return nil, 0, err
	}
	return profiles, total, nil
}
