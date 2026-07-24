package message

import "context"

type Repository interface {
	Create(ctx context.Context, obj *Message) error
	// CreateIdempotent вставляет сообщение с учётом идемпотентного ключа
	// ClientMessageID. Возвращает created=true, если строка реально вставлена;
	// created=false, если такой ключ уже существовал — тогда obj заполняется
	// исходным сообщением. При пустом ключе всегда вставляет (created=true).
	CreateIdempotent(ctx context.Context, obj *Message) (created bool, err error)
	Delete(ctx context.Context, id uint) error
	Update(ctx context.Context, obj *Message) error
	GetMessages(ctx context.Context, filter Filter) ([]*Message, error)
	GetByID(ctx context.Context, id uint) (*Message, error)
}
