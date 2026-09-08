package personal

import "context"

type Repository interface {
	Create(ctx context.Context, obj *Model) error
	List(ctx context.Context, userID uint, since *uint, upTo uint, limit int) (Changes, error)
}

type RevisionRepository interface {
	UpsertRevision(ctx context.Context, userID uint) (uint, error)
	GetRevision(ctx context.Context, userID uint) (uint, error)
}
