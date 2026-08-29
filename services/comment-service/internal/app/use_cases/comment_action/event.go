package comment_action

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/services/comment-service/internal/domain/comment"
)

// EventPublisher — порт для публикации доменных событий комментариев во внешнюю шину.
// Интерфейс объявлен на стороне потребителя; kafka.CommentPublisher удовлетворяет его.
type EventPublisher interface {
	PublishCommentCreated(ctx context.Context, c *comment.Comment) error
}

// NoOpEventPublisher — заглушка на случай выключенной шины.
type NoOpEventPublisher struct{}

func (NoOpEventPublisher) PublishCommentCreated(context.Context, *comment.Comment) error {
	return nil
}
