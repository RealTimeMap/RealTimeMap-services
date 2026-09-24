package bug

import (
	"context"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Тесты проверки бага разработчиком и её связи с привязкой к задаче.
// Репозиторий — фейк в памяти, БД не нужна.

type fakeRepo struct {
	bugs map[uint]*Model
}

func newFakeRepo(bugs ...Model) *fakeRepo {
	r := &fakeRepo{bugs: map[uint]*Model{}}
	for i := range bugs {
		b := bugs[i]
		r.bugs[b.ID] = &b
	}
	return r
}

func (r *fakeRepo) Create(_ context.Context, data *Model) error {
	r.bugs[data.ID] = data
	return nil
}

func (r *fakeRepo) GetByID(_ context.Context, id uint) (*Model, error) {
	b, ok := r.bugs[id]
	if !ok {
		return nil, ErrBugNotFound(id)
	}
	cp := *b
	return &cp, nil
}

func (r *fakeRepo) GetByTaskID(_ context.Context, taskID uint) (*Model, error) {
	for _, b := range r.bugs {
		if b.IsLinkedTo(taskID) {
			cp := *b
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeRepo) GetList(context.Context, Filter) ([]Model, error) { return nil, nil }

func (r *fakeRepo) Update(_ context.Context, data *Model) error {
	cp := *data
	r.bugs[data.ID] = &cp
	return nil
}

var fixedNow = time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

func newTestService(repo Repository) *Service {
	s := NewService(repo, zap.NewNop())
	s.now = func() time.Time { return fixedNow }
	return s
}

func bugWith(id uint, status Status) Model {
	m := Model{Status: status}
	m.ID = id
	return m
}

func requireConflict(t *testing.T, err error) {
	t.Helper()
	var conflict *apperror.ConflictError
	require.ErrorAs(t, err, &conflict)
}

func TestConfirm(t *testing.T) {
	ctx := context.Background()

	t.Run("новый баг подтверждается и получает отметку проверки", func(t *testing.T) {
		repo := newFakeRepo(bugWith(1, New))
		s := newTestService(repo)

		obj, first, err := s.Confirm(ctx, ConfirmParams{BugID: 1, Comment: "воспроизвёлся на Android 14"})
		require.NoError(t, err)

		assert.True(t, first, "первое подтверждение")
		assert.Equal(t, Confirmed, obj.Status)
		saved := repo.bugs[1]
		assert.Equal(t, Confirmed, saved.Status)
		require.NotNil(t, saved.Review.At)
		assert.Equal(t, fixedNow, *saved.Review.At)
		assert.Equal(t, "воспроизвёлся на Android 14", saved.Review.Comment)
		assert.Empty(t, saved.Review.RejectReason)
		require.NotNil(t, saved.FirstConfirmedAt)
		assert.Equal(t, fixedNow, *saved.FirstConfirmedAt)
	})

	t.Run("повторное подтверждение ничего не меняет", func(t *testing.T) {
		b := bugWith(1, Confirmed)
		b.Review = Review{At: utils.Ptr(fixedNow.Add(-time.Hour)), Comment: "первое"}
		repo := newFakeRepo(b)

		obj, first, err := newTestService(repo).Confirm(ctx, ConfirmParams{BugID: 1, Comment: "второе"})
		require.NoError(t, err)
		assert.False(t, first, "повтор не даёт второй награды")
		assert.Equal(t, "первое", obj.Review.Comment)
	})

	for _, status := range []Status{Rejected, InWork, Closed, Canceled} {
		t.Run("нельзя подтвердить баг в статусе "+string(status), func(t *testing.T) {
			repo := newFakeRepo(bugWith(1, status))

			_, first, err := newTestService(repo).Confirm(ctx, ConfirmParams{BugID: 1})
			requireConflict(t, err)
			assert.False(t, first)
			assert.Equal(t, status, repo.bugs[1].Status)
		})
	}
}

func TestConfirmAfterReopenIsNotFirst(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo(bugWith(1, New))
	s := newTestService(repo)

	_, first, err := s.Confirm(ctx, ConfirmParams{BugID: 1})
	require.NoError(t, err)
	require.True(t, first)

	_, err = s.Reopen(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, repo.bugs[1].FirstConfirmedAt, "возврат на проверку не стирает первое подтверждение")

	_, first, err = s.Confirm(ctx, ConfirmParams{BugID: 1})
	require.NoError(t, err)
	assert.False(t, first, "повторное подтверждение после reopen не даёт второй награды")
}

func TestReject(t *testing.T) {
	ctx := context.Background()

	t.Run("непроверенный баг отклоняется с причиной", func(t *testing.T) {
		repo := newFakeRepo(bugWith(1, New))

		obj, err := newTestService(repo).Reject(ctx, RejectParams{
			BugID: 1, Reason: ReasonNotReproducible, Comment: "три устройства, не повторяется",
		})
		require.NoError(t, err)

		assert.Equal(t, Rejected, obj.Status)
		saved := repo.bugs[1]
		assert.Equal(t, ReasonNotReproducible, saved.Review.RejectReason)
		assert.Equal(t, "три устройства, не повторяется", saved.Review.Comment)
		require.NotNil(t, saved.Review.At)
	})

	t.Run("подтверждённый, но свободный баг тоже можно отклонить", func(t *testing.T) {
		repo := newFakeRepo(bugWith(1, Confirmed))

		obj, err := newTestService(repo).Reject(ctx, RejectParams{BugID: 1, Reason: ReasonDuplicate})
		require.NoError(t, err)
		assert.Equal(t, Rejected, obj.Status)
	})

	t.Run("неизвестная причина — ошибка валидации", func(t *testing.T) {
		repo := newFakeRepo(bugWith(1, New))

		_, err := newTestService(repo).Reject(ctx, RejectParams{BugID: 1, Reason: "whatever"})
		var validation *apperror.FieldValidationError
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, New, repo.bugs[1].Status)
	})

	t.Run("баг, привязанный к задаче, отклонить нельзя", func(t *testing.T) {
		b := bugWith(1, Confirmed)
		b.TaskID = utils.Ptr(uint(7))
		repo := newFakeRepo(b)

		_, err := newTestService(repo).Reject(ctx, RejectParams{BugID: 1, Reason: ReasonNotABug})
		requireConflict(t, err)
	})

	for _, status := range []Status{InWork, Closed, Canceled} {
		t.Run("нельзя отклонить баг в статусе "+string(status), func(t *testing.T) {
			repo := newFakeRepo(bugWith(1, status))

			_, err := newTestService(repo).Reject(ctx, RejectParams{BugID: 1, Reason: ReasonNotABug})
			requireConflict(t, err)
		})
	}
}

func TestReopen(t *testing.T) {
	ctx := context.Background()

	t.Run("отклонённый баг возвращается на проверку без прежнего решения", func(t *testing.T) {
		b := bugWith(1, Rejected)
		b.Review = Review{At: utils.Ptr(fixedNow), RejectReason: ReasonSpam, Comment: "спам"}
		repo := newFakeRepo(b)

		obj, err := newTestService(repo).Reopen(ctx, 1)
		require.NoError(t, err)

		assert.Equal(t, New, obj.Status)
		assert.Equal(t, Review{}, repo.bugs[1].Review)
	})

	t.Run("баг в работе на проверку не вернуть", func(t *testing.T) {
		b := bugWith(1, InWork)
		b.TaskID = utils.Ptr(uint(7))
		repo := newFakeRepo(b)

		_, err := newTestService(repo).Reopen(ctx, 1)
		requireConflict(t, err)
	})
}

func TestLinkRequiresConfirmation(t *testing.T) {
	ctx := context.Background()

	t.Run("непроверенный баг в задачу не берётся", func(t *testing.T) {
		repo := newFakeRepo(bugWith(1, New))

		_, err := newTestService(repo).Link(ctx, LinkParams{BugID: 1, TaskID: 7})
		requireConflict(t, err)
		assert.Nil(t, repo.bugs[1].TaskID)
	})

	t.Run("отклонённый баг в задачу не берётся", func(t *testing.T) {
		repo := newFakeRepo(bugWith(1, Rejected))

		_, err := newTestService(repo).Link(ctx, LinkParams{BugID: 1, TaskID: 7})
		requireConflict(t, err)
	})

	t.Run("подтверждённый баг берётся в задачу и переходит в работу", func(t *testing.T) {
		repo := newFakeRepo(bugWith(1, Confirmed))

		obj, err := newTestService(repo).Link(ctx, LinkParams{BugID: 1, TaskID: 7})
		require.NoError(t, err)
		assert.Equal(t, InWork, obj.Status)
		assert.True(t, repo.bugs[1].IsLinkedTo(7))
	})

	t.Run("снятый с задачи баг остаётся подтверждённым", func(t *testing.T) {
		b := bugWith(1, InWork)
		b.TaskID = utils.Ptr(uint(7))
		repo := newFakeRepo(b)

		obj, err := newTestService(repo).Unlink(ctx, 7)
		require.NoError(t, err)
		assert.Equal(t, Confirmed, obj.Status)
		assert.Nil(t, repo.bugs[1].TaskID)
	})
}

func TestSyncStatusKeepsReview(t *testing.T) {
	ctx := context.Background()

	t.Run("возврат задачи в new не отправляет баг на повторную проверку", func(t *testing.T) {
		b := bugWith(1, InWork)
		b.TaskID = utils.Ptr(uint(7))
		repo := newFakeRepo(b)

		obj, err := newTestService(repo).SyncStatus(ctx, SyncParams{TaskID: 7, Status: New})
		require.NoError(t, err)
		assert.Equal(t, Confirmed, obj.Status)
	})

	t.Run("задача не может отклонить баг", func(t *testing.T) {
		b := bugWith(1, InWork)
		b.TaskID = utils.Ptr(uint(7))
		repo := newFakeRepo(b)

		_, err := newTestService(repo).SyncStatus(ctx, SyncParams{TaskID: 7, Status: Rejected})
		var validation *apperror.FieldValidationError
		require.ErrorAs(t, err, &validation)
		assert.Equal(t, InWork, repo.bugs[1].Status)
	})
}
