package reaction

import "context"

type Repository interface {
	Like(ctx context.Context, commentID, userID uint) error
	UnLike(ctx context.Context, commentID, userID uint) error
	Count(ctx context.Context, commentID uint) (int64, error)
	IsLiked(ctx context.Context, commentID, userID uint) (bool, error)
	// LikedByUser возвращает подмножество commentIDs, которые пользователь лайкнул.
	// Одним запросом на всю страницу — чтобы не делать IsLiked в цикле (N+1).
	LikedByUser(ctx context.Context, commentIDs []uint, userID uint) (map[uint]bool, error)
}
