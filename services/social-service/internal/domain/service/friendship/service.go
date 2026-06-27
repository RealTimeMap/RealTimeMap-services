package friendship

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/domainerrors"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
)

type Service struct {
	repo        repository.FriendShipRepository
	profileRepo repository.ProfileRepository
	blockedRepo repository.BlockedUserRepository

	logger *zap.Logger
}

func NewService(
	repo repository.FriendShipRepository,
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

// SendRequest отправляет запрос в друзья от userID к friendID.
func (s *Service) SendRequest(ctx context.Context, userID, friendID uint) error {
	if userID == friendID {
		return domainerrors.CantFriendYourSelf(friendID)
	}

	if err := s.checkProfileExists(ctx, friendID); err != nil {
		return err
	}

	if err := s.checkNotBlocked(ctx, userID, friendID); err != nil {
		return err
	}

	relation, err := s.repo.GetRelation(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if relation != nil {
		if relation.Status == model.Accepted {
			return domainerrors.AlreadyFriends(friendID)
		}
		return domainerrors.RequestAlreadyExists(friendID)
	}

	return s.repo.SendRequest(ctx, userID, friendID)
}

// AcceptRequest принимает входящий запрос. userID — получатель, friendID — отправитель.
func (s *Service) AcceptRequest(ctx context.Context, userID, friendID uint) error {
	relation, err := s.repo.GetRelation(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if relation == nil || relation.UserID != friendID || relation.Status != model.Waiting {
		return domainerrors.FriendRequestNotFound(friendID)
	}

	return s.repo.AcceptRequest(ctx, userID, friendID)
}

// DeclineRequest отклоняет входящий запрос. userID — получатель, friendID — отправитель.
func (s *Service) DeclineRequest(ctx context.Context, userID, friendID uint) error {
	relation, err := s.repo.GetRelation(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if relation == nil || relation.UserID != friendID || relation.Status != model.Waiting {
		return domainerrors.FriendRequestNotFound(friendID)
	}

	return s.repo.DeclineRequest(ctx, userID, friendID)
}

// Remove удаляет существующую дружбу между userID и friendID.
func (s *Service) Remove(ctx context.Context, userID, friendID uint) error {
	relation, err := s.repo.GetRelation(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if relation == nil {
		return domainerrors.FriendshipNotFound(friendID)
	}

	return s.repo.Remove(ctx, userID, friendID)
}

// GetFriendsProfile возвращает профили друзей пользователя с пагинацией.
func (s *Service) GetFriendsProfile(ctx context.Context, userID uint, params *FriendsSearchParams) ([]*model.Profile, int64, error) {
	params.Pagination.Defaults()

	ids, err := s.repo.GetFriends(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	total := int64(len(ids))
	pageIDs := paginateIDs(ids, params.Pagination.Offset(), params.Pagination.Limit())
	if len(pageIDs) == 0 {
		return []*model.Profile{}, total, nil
	}

	profiles, err := s.profileRepo.GetProfilesByIDs(ctx, pageIDs)
	if err != nil {
		return nil, 0, err
	}

	return profiles, total, nil
}

func (s *Service) checkProfileExists(ctx context.Context, userID uint) error {
	_, err := s.profileRepo.GetProfile(ctx, userID)
	return err
}

func (s *Service) checkNotBlocked(ctx context.Context, userID, friendID uint) error {
	blocked, err := s.blockedRepo.ExistsBetween(ctx, userID, friendID)
	if err != nil {
		return err
	}
	if blocked {
		return domainerrors.FriendshipBlocked()
	}
	return nil
}

func paginateIDs(ids []uint, offset, limit int) []uint {
	if offset >= len(ids) {
		return nil
	}
	end := offset + limit
	if end > len(ids) {
		end = len(ids)
	}
	return ids[offset:end]
}
