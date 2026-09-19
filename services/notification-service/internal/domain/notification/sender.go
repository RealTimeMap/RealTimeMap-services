package notification

import "context"

type Message struct {
	Token   string
	Title   string
	Content string
	Data    map[string]string
}

type Sender interface {
	Send(ctx context.Context, mess Message) error
}
