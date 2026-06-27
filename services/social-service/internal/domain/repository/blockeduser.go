package repository

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/model"
)

type BlockedUserRepository interface {
	Block(ctx context.Context, userID, blockedUserID uint) (bool, error)
	Unblock(ctx context.Context, data *model.BlockedUser) error
	GetBlockedUsers(ctx context.Context, userID uint, params pagination.Params) ([]uint, int64, error)
	GetByID(ctx context.Context, userID uint, blockedUserID uint) (*model.BlockedUser, error)
	// ExistsBetween проверяет, заблокировал ли кто-либо из двух пользователей другого (в любую сторону)
	ExistsBetween(ctx context.Context, userID, otherID uint) (bool, error)
}
