package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
)

type FriendShipRepository interface {
	// SendRequest отправка запроса в друзья
	SendRequest(ctx context.Context, userID, friendID uint) error
	// AcceptRequest Принять запрос в друзья
	AcceptRequest(ctx context.Context, userID, friendID uint) error
	// DeclineRequest Отклонить запрос в друзья
	DeclineRequest(ctx context.Context, userID, friendID uint) error
	// Remove Удалить из друзей
	Remove(ctx context.Context, userID, friendID uint) error
	// GetFriends получить всех друзей (только id)
	GetFriends(ctx context.Context, userID uint) ([]uint, error)
	// CountFriends число друзей
	CountFriends(ctx context.Context, userID uint, status model.FriendshipStatus) (int64, error)
	// CountFriendAndSubs количество друзей и подписчиков
	CountFriendAndSubs(ctx context.Context, userID uint) (int64, int64, error)
}
