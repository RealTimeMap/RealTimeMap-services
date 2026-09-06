package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
)

type SubscriptionRepository interface {
	// Subscribe создаёт подписку subscriberID -> targetID.
	// Возвращает false, если подписка уже существовала.
	Subscribe(ctx context.Context, subscriberID, targetID uint) (bool, error)
	// Unsubscribe удаляет подписку subscriberID -> targetID
	Unsubscribe(ctx context.Context, subscriberID, targetID uint) error
	// Exists проверяет наличие подписки subscriberID -> targetID
	Exists(ctx context.Context, subscriberID, targetID uint) (bool, error)
	// GetSubscriptions id профилей, на которые подписан userID (исходящие)
	GetSubscriptions(ctx context.Context, userID uint, params pagination.Params) ([]uint, int64, error)
	// GetSubscribers id профилей, подписанных на userID (входящие)
	GetSubscribers(ctx context.Context, userID uint, params pagination.Params) ([]uint, int64, error)
	// CountSubscribers число подписчиков userID (входящие)
	CountSubscribers(ctx context.Context, userID uint) (int64, error)
	// CountSubscriptions число подписок userID (исходящие)
	CountSubscriptions(ctx context.Context, userID uint) (int64, error)
	// DeleteBetween удаляет подписки между двумя пользователями в обе стороны.
	// Используется при блокировке: заблокированный не должен оставаться подписчиком.
	DeleteBetween(ctx context.Context, userID, otherID uint) error
}
