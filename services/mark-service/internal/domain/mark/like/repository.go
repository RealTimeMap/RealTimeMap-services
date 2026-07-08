package like

import "context"

type Repository interface {
	Like(ctx context.Context, markID, userID uint) error
	UnLike(ctx context.Context, markID, userID uint) error
	Count(ctx context.Context, markID uint) (int64, error)
	IsLiked(ctx context.Context, markID, userID uint) (bool, error)
}
