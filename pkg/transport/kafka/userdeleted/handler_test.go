package userdeleted

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

type fakeAccounts struct {
	erased []uint
	err    error
}

func (f *fakeAccounts) EraseUser(_ context.Context, userID uint) error {
	f.erased = append(f.erased, userID)
	return f.err
}

func TestHandleMessage(t *testing.T) {
	deleted := kafka.Message{Value: []byte(`{"type":"user.deleted","payload":{"user_id":42}}`)}

	t.Run("user.deleted обезличивает", func(t *testing.T) {
		repo := &fakeAccounts{}
		require.NoError(t, Handler(repo, zap.NewNop())(context.Background(), deleted))
		assert.Equal(t, []uint{42}, repo.erased)
	})

	t.Run("остальные события топика пропускаются", func(t *testing.T) {
		repo := &fakeAccounts{}
		msg := kafka.Message{Value: []byte(`{"type":"user.logged_in","payload":{"user_id":42}}`)}
		require.NoError(t, Handler(repo, zap.NewNop())(context.Background(), msg))
		assert.Empty(t, repo.erased)
	})

	t.Run("сбой БД — повтор", func(t *testing.T) {
		err := Handler(&fakeAccounts{err: errors.New("db down")}, zap.NewNop())(
			context.Background(), deleted)
		assert.ErrorIs(t, err, consumer.ErrRetryable)
	})

	t.Run("битое тело — пропуск", func(t *testing.T) {
		err := Handler(&fakeAccounts{}, zap.NewNop())(
			context.Background(), kafka.Message{Value: []byte(`{`)})
		assert.ErrorIs(t, err, consumer.ErrSkip)
	})
}
