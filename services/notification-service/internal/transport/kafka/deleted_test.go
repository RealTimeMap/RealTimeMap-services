package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	segmentio "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// stubEraser запоминает, чьи устройства просили удалить.
type stubEraser struct {
	users []uint
	err   error
}

func (s *stubEraser) DeleteUser(_ context.Context, userID uint) error {
	s.users = append(s.users, userID)
	return s.err
}

func TestHandleUserDeleted(t *testing.T) {
	msg := segmentio.Message{Value: []byte(
		`{"type":"user.deleted","payload":{"user_id":42,"email":"a@b.c"}}`)}

	t.Run("удаляет устройства и ничего не отправляет", func(t *testing.T) {
		notifier, eraser := &notifierStub{}, &stubEraser{}
		h := NewHandler(notifier, eraser, zap.NewNop())

		if err := h.HandleMessage(context.Background(), msg); err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		if len(eraser.users) != 1 || eraser.users[0] != 42 {
			t.Fatalf("erased %v, want [42]", eraser.users)
		}
		if len(notifier.cmds) != 0 {
			t.Fatalf("отправлено %d уведомлений, want 0", len(notifier.cmds))
		}
	})

	t.Run("сбой БД — повтор", func(t *testing.T) {
		h := NewHandler(&notifierStub{}, &stubEraser{err: errors.New("db down")}, zap.NewNop())

		if err := h.HandleMessage(context.Background(), msg); !errors.Is(err, consumer.ErrRetryable) {
			t.Fatalf("err = %v, want retryable", err)
		}
	})
}
