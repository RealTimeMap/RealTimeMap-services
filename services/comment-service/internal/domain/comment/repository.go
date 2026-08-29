package comment

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, comment *Comment) (*Comment, error)
	GetByID(ctx context.Context, id uint) (*Comment, error)
	GetComments(ctx context.Context, filters CommentFilter) ([]*Comment, bool, error)
	Update(ctx context.Context, comment *Comment) (*Comment, error)
}
