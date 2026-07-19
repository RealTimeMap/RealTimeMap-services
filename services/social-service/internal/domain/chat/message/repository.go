package message

import "context"

type Repository interface {
	Create(ctx context.Context, obj *Message) error
	Delete(ctx context.Context, id uint) error
	Update(ctx context.Context, obj *Message) error
	GetMessages(ctx context.Context, filter Filter) ([]*Message, error)
	GetByID(ctx context.Context, id uint) (*Message, error)
}
