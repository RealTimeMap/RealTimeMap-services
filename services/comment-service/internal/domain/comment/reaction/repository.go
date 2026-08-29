package reaction

import "context"

type Repository interface {
	Like(ctx context.Context, commentID, userID uint) error
	UnLike(ctx context.Context, commentID, userID uint) error
	Count(ctx context.Context, commentID uint) (int64, error)
	IsLiked(ctx context.Context, commentID, userID uint) (bool, error)
}
