package bug

import (
	"context"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/feedback-service/internal/domain/bug"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Тесты публикации bug.confirmed. Домен — фейк: здесь проверяется только
// решение «слать или нет», переходы статусов покрыты в domain/bug.

type fakeReviewer struct {
	confirmed *bug.Model
	first     bool
}

func (f *fakeReviewer) Confirm(context.Context, bug.ConfirmParams) (*bug.Model, bool, error) {
	cp := *f.confirmed
	return &cp, f.first, nil
}

func (f *fakeReviewer) Reject(context.Context, bug.RejectParams) (*bug.Model, error) { return nil, nil }
func (f *fakeReviewer) Reopen(context.Context, uint) (*bug.Model, error)             { return nil, nil }

type fakePublisher struct {
	published chan *bug.Model
}

func newFakePublisher() *fakePublisher {
	return &fakePublisher{published: make(chan *bug.Model, 1)}
}

func (f *fakePublisher) PublishBugConfirmed(_ context.Context, b *bug.Model) error {
	f.published <- b
	return nil
}

// Публикация фоновая, поэтому ждём её с таймаутом, а отсутствие —
// короткой паузой.
const (
	publishWait   = time.Second
	noPublishWait = 100 * time.Millisecond
)

func confirmedBug(userID *uint) *bug.Model {
	b := &bug.Model{UserID: userID, Status: bug.Confirmed, Tag: bug.TagUI}
	b.ID = 5
	return b
}

func TestConfirmPublishesEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("первое подтверждение бага с автором уходит в шину", func(t *testing.T) {
		pub := newFakePublisher()
		h := NewReviewBugHandler(&fakeReviewer{confirmed: confirmedBug(utils.Ptr(uint(42))), first: true}, pub, zap.NewNop())

		_, err := h.Confirm(ctx, ConfirmBugCommand{BugID: 5})
		require.NoError(t, err)

		select {
		case b := <-pub.published:
			assert.Equal(t, uint(5), b.ID)
			require.NotNil(t, b.UserID)
			assert.Equal(t, uint(42), *b.UserID)
		case <-time.After(publishWait):
			t.Fatal("событие bug.confirmed не опубликовано")
		}
	})

	t.Run("анонимный отчёт события не шлёт", func(t *testing.T) {
		pub := newFakePublisher()
		h := NewReviewBugHandler(&fakeReviewer{confirmed: confirmedBug(nil), first: true}, pub, zap.NewNop())

		_, err := h.Confirm(ctx, ConfirmBugCommand{BugID: 5})
		require.NoError(t, err)

		select {
		case <-pub.published:
			t.Fatal("событие для отчёта без автора опубликовано")
		case <-time.After(noPublishWait):
		}
	})

	t.Run("повторное подтверждение события не шлёт", func(t *testing.T) {
		pub := newFakePublisher()
		h := NewReviewBugHandler(&fakeReviewer{confirmed: confirmedBug(utils.Ptr(uint(42))), first: false}, pub, zap.NewNop())

		_, err := h.Confirm(ctx, ConfirmBugCommand{BugID: 5})
		require.NoError(t, err)

		select {
		case <-pub.published:
			t.Fatal("повторное подтверждение опубликовало событие")
		case <-time.After(noPublishWait):
		}
	})

	t.Run("без publisher обработчик работает на заглушке", func(t *testing.T) {
		h := NewReviewBugHandler(&fakeReviewer{confirmed: confirmedBug(utils.Ptr(uint(42))), first: true}, nil, zap.NewNop())

		_, err := h.Confirm(ctx, ConfirmBugCommand{BugID: 5})
		require.NoError(t, err)
	})
}
