package repository

import "context"

type AccrualRepository interface {
	IncShare(ctx context.Context, markID uint) (int64, error)
	Like(ctx context.Context, markID, userID uint) error
	UnLike(ctx context.Context, markID, userID uint) error
}
