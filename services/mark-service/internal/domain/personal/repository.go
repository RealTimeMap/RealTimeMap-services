package personal

import "context"

type Repository interface {
	Create(ctx context.Context, obj *Model) error
}

type RevisionRepository interface {
	UpsertRevision(ctx context.Context, userID uint) (uint, error)
	GetRevision(ctx context.Context, userID uint) (uint, error)
}
