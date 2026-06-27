package postgres

import (
	"context"
	"errors"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// GetRelation возвращает связь между двумя пользователями в любом направлении (nil если связи нет)
func (r *PgFriendshipRepository) GetRelation(ctx context.Context, userID, friendID uint) (*model.Friendship, error) {
	var relation model.Friendship
	err := r.db.WithContext(ctx).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, friendID, friendID, userID).
		First(&relation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &relation, nil
}

// SendRequest отправка запроса в друзья. Создаёт строку userID -> friendID со статусом waiting.
func (r *PgFriendshipRepository) SendRequest(ctx context.Context, userID, friendID uint) error {
	payload := &model.Friendship{
		UserID:   userID,
		FriendID: friendID,
		Status:   model.Waiting,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(payload).Error
}

// AcceptRequest Принять запрос в друзья. userID — получатель, friendID — отправитель запроса.
func (r *PgFriendshipRepository) AcceptRequest(ctx context.Context, userID, friendID uint) error {
	return r.db.WithContext(ctx).Model(&model.Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = ?", friendID, userID, model.Waiting).
		Update("status", model.Accepted).Error
}

// DeclineRequest Отклонить запрос в друзья. userID — получатель, friendID — отправитель запроса.
func (r *PgFriendshipRepository) DeclineRequest(ctx context.Context, userID, friendID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND friend_id = ? AND status = ?", friendID, userID, model.Waiting).
		Delete(&model.Friendship{}).Error
}

// Remove Удалить из друзей. Удаляет связь в любом направлении.
func (r *PgFriendshipRepository) Remove(ctx context.Context, userID, friendID uint) error {
	return r.db.WithContext(ctx).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
			userID, friendID, friendID, userID).
		Delete(&model.Friendship{}).Error
}

// GetFriends получить всех друзей (только id). Ищет связи в обе стороны.
func (r *PgFriendshipRepository) GetFriends(ctx context.Context, userID uint) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&model.Friendship{}).
		Where("status = ?", model.Accepted).
		Where("user_id = ? OR friend_id = ?", userID, userID).
		Select("CASE WHEN user_id = ? THEN friend_id ELSE user_id END", userID).
		Scan(&ids).Error
	return ids, err
}

// CountFriends число друзей с заданным статусом. Учитывает связи в обе стороны.
func (r *PgFriendshipRepository) CountFriends(ctx context.Context, userID uint, status model.FriendshipStatus) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).Model(&model.Friendship{}).
		Where("user_id = ? OR friend_id = ?", userID, userID).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// CountFriendAndSubs возвращает количество друзей (accepted, обе стороны)
// и подписчиков (входящие waiting-запросы, где userID — получатель).
func (r *PgFriendshipRepository) CountFriendAndSubs(ctx context.Context, userID uint) (int64, int64, error) {
	var friendCount, subsCount int64
	err := r.db.WithContext(ctx).Model(&model.Friendship{}).
		Where("user_id = ? OR friend_id = ?", userID, userID).
		Select(
			"COUNT(*) FILTER (WHERE status = ?) AS friend_count, "+
				"COUNT(*) FILTER (WHERE status = ? AND friend_id = ?) AS subs_count",
			model.Accepted,
			model.Waiting,
			userID,
		).
		Row().
		Scan(&friendCount, &subsCount)

	return friendCount, subsCount, err
}
