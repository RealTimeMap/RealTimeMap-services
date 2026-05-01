package profile

import (
	"bytes"
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// ProgressGetter интерфейс для получения геймификационного прогресса
type ProgressGetter interface {
	GetUserProgress(ctx context.Context, userID uint) (*model.Progress, error)
}

const avatarMaxSize = 5 * 1024 * 1024 // 5MB

type Service struct {
	profileRepo    repository.ProfileRepository
	store          storage.Storage
	photoValidator *mediavalidator.PhotoValidator
	progress       ProgressGetter

	logger *zap.Logger
}

func NewProfileService(
	profileRepo repository.ProfileRepository,
	store storage.Storage,
	photoValidator *mediavalidator.PhotoValidator,
	progress ProgressGetter,
	logger *zap.Logger,
) *Service {
	return &Service{
		profileRepo:    profileRepo,
		store:          store,
		photoValidator: photoValidator,
		progress:       progress,
		logger:         logger,
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

func (s *Service) UpdateProfile(ctx context.Context, in UpdateProfileInput) (*model.Profile, error) {
	s.logger.Info("ProfileService.UpdateProfile", zap.Uint("user_id", in.UserID))

	current, err := s.profileRepo.GetProfile(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}
	if in.Username != nil {
		fields["username"] = *in.Username
	}
	if in.Tag != nil {
		fields["tag"] = *in.Tag
	}

	var (
		uploadedAvatarKey string
		oldAvatarKey      string
	)
	if in.Avatar != nil {
		if err := s.photoValidator.ValidateSinglePhoto(mediavalidator.PhotoInput{
			Data:     in.Avatar.Data,
			FileName: in.Avatar.FileName,
		}); err != nil {
			return nil, err
		}

		photo, err := s.store.Upload(ctx, bytes.NewReader(in.Avatar.Data), storage.UploadOptions{
			FileName:      in.Avatar.FileName,
			Category:      storage.CategoryProfileAvatar,
			MaxSize:       avatarMaxSize,
			Optimize:      true,
			GenerateThumb: true,
			ThumbWidth:    200,
			ThumbHeight:   200,
		})
		if err != nil {
			return nil, err
		}

		fields["avatar"] = *photo
		uploadedAvatarKey = photo.StorageKey
		oldAvatarKey = current.Avatar.StorageKey
	}

	if len(fields) == 0 {
		return current, nil
	}

	updated, err := s.profileRepo.Update(ctx, in.UserID, fields)
	if err != nil {
		if uploadedAvatarKey != "" {
			if delErr := s.store.Delete(ctx, uploadedAvatarKey); delErr != nil {
				s.logger.Warn("failed to rollback uploaded avatar",
					zap.String("storage_key", uploadedAvatarKey), zap.Error(delErr))
			}
		}
		return nil, err
	}

	if oldAvatarKey != "" {
		go func(key string) {
			if err := s.store.Delete(context.Background(), key); err != nil {
				s.logger.Warn("failed to delete old avatar",
					zap.String("storage_key", key), zap.Error(err))
			}
		}(oldAvatarKey)
	}

	return updated, nil
}

func (s *Service) GetProfile(ctx context.Context, userId uint) (*model.Profile, error) {
	s.logger.Info("ProfileService.GetProfile", zap.Uint("user_id", userId))
	profile, err := s.profileRepo.GetProfile(ctx, userId)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// GetMyProfile возвращает профиль текущего пользователя вместе с прогрессом
// Прогресс необязателен — если gamification-service недоступен, поле остаётся nil.
func (s *Service) GetMyProfile(ctx context.Context, userID uint) (*model.Profile, *model.Progress, error) {
	s.logger.Info("ProfileService.GetMyProfile", zap.Uint("user_id", userID))

	var (
		profile  *model.Profile
		progress *model.Progress
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		p, err := s.profileRepo.GetProfile(gCtx, userID)
		if err != nil {
			return err
		}
		profile = p
		return nil
	})

	g.Go(func() error {
		if s.progress == nil {
			return nil
		}
		p, err := s.progress.GetUserProgress(gCtx, userID)
		if err != nil {
			s.logger.Warn("gamification fetch failed",
				zap.Uint("user_id", userID), zap.Error(err))
			return nil
		}
		progress = p
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return profile, progress, nil
}

func (s *Service) GetProfilesByIDs(ctx context.Context, ids []uint) ([]*model.Profile, error) {
	s.logger.Info("ProfileService.GetProfilesByIDs", zap.Int("ids_count", len(ids)))
	if len(ids) == 0 {
		return []*model.Profile{}, nil
	}
	return s.profileRepo.GetProfilesByIDs(ctx, ids)
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
