package accrual

import "context"

type Repository interface {
	IncShare(ctx context.Context, markID uint) (int64, error)
	Like(ctx context.Context, markID, userID uint) error
	UnLike(ctx context.Context, markID, userID uint) error
}
