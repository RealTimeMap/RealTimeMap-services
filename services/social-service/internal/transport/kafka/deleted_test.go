package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/consumer"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeEraser struct {
	calls []uint
	err   error
}

func (f *fakeEraser) DeleteUser(_ context.Context, userID uint) error {
	f.calls = append(f.calls, userID)
	return f.err
}

func TestHandleMessageUserDeleted(t *testing.T) {
	msg := kafka.Message{Value: []byte(
		`{"type":"user.deleted","payload":{"user_id":42,"email":"a@b.c"}}`)}

	t.Run("стирает данные пользователя", func(t *testing.T) {
		eraser := &fakeEraser{}
		h := NewHandler(nil, eraser, zap.NewNop())

		require.NoError(t, h.HandleMessage(context.Background(), msg))
		assert.Equal(t, []uint{42}, eraser.calls)
	})

	t.Run("сбой хранилища — повтор", func(t *testing.T) {
		h := NewHandler(nil, &fakeEraser{err: errors.New("db down")}, zap.NewNop())

		err := h.HandleMessage(context.Background(), msg)
		assert.ErrorIs(t, err, consumer.ErrRetryable)
	})

	t.Run("без user_id — пропуск, удалять некого", func(t *testing.T) {
		eraser := &fakeEraser{}
		h := NewHandler(nil, eraser, zap.NewNop())

		err := h.HandleMessage(context.Background(), kafka.Message{Value: []byte(
			`{"type":"user.deleted","payload":{"email":"a@b.c"}}`)})
		assert.ErrorIs(t, err, consumer.ErrSkip)
		assert.Empty(t, eraser.calls)
	})
}
