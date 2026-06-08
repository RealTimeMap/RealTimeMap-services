package postgres

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PgFriendshipRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgFriendshipRepository(db *gorm.DB, logger *zap.Logger) repository.FriendShipRepository {
	return &PgFriendshipRepository{
		db:     db,
		logger: logger,
	}
}

func (r *PgFriendshipRepository) SendRequest(ctx context.Context, userID, friendID uint) error {
	panic("implement me")
}

// AcceptRequest Принять запрос в друзья
func (r *PgFriendshipRepository) AcceptRequest(ctx context.Context, userID, friendID uint) error {
	panic("implement me")
}

// DeclineRequest Отклонить запрос в друзья
func (r *PgFriendshipRepository) DeclineRequest(ctx context.Context, userID, friendID uint) error {
	panic("implement me")
}

// Remove Удалить из друзей
func (r *PgFriendshipRepository) Remove(ctx context.Context, userID, friendID uint) error {
	panic("implement me")
}

// GetFriends получить всех друзей (только id)
func (r *PgFriendshipRepository) GetFriends(ctx context.Context, userID uint) ([]uint, error) {
	panic("implement me")
}

// CountFriends число друзей
func (r *PgFriendshipRepository) CountFriends(ctx context.Context, userID uint, status model.FriendshipStatus) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.Friendship{}).
		Where("user_id = ?", userID).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

func (r *PgFriendshipRepository) CountFriendAndSubs(ctx context.Context, userID uint) (int64, int64, error) {
	var friendCount, subsCount int64
	err := r.db.WithContext(ctx).Model(&model.Friendship{}).
		Where("user_id = ?", userID).
		Select(
			"COUNT(*) FILTER (WHERE status = ?) AS friend_count, "+
				"COUNT(*) FILTER (WHERE status = ?) AS subs_count",
			model.Accepted,
			model.Waiting,
		).
		Row().
		Scan(&friendCount, &subsCount)

	return friendCount, subsCount, err
}
