package mark

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тесты фиксируют контракт DeleteMark: метка читается, проверяется владелец,
// и только после этого запись удаляется через репозиторий.
//
// Ранее метод возвращал nil, ни разу не вызвав markRepo.Delete, — клиент по
// маршруту DELETE /api/v2/marks/:markID получал 204, а метка оставалась в БД.
// Проверка deleteCalls ниже существует именно для того, чтобы эта регрессия не
// повторилась незамеченной.

func TestDeleteMark(t *testing.T) {
	ctx := context.Background()
	const ownerID, markID = uint(42), uint(100)

	liveMark := func(userID uint) *Mark {
		return &Mark{UserID: userID}
	}

	t.Run("владелец удаляет свою метку", func(t *testing.T) {
		markRepo := &fakeMarkRepo{getByIDResult: liveMark(ownerID)}
		s := newServiceWith(markRepo, nil, nil)

		err := s.DeleteMark(ctx, ownerID, markID)

		require.NoError(t, err)
		assert.Equal(t, 1, markRepo.deleteCalls, "удаление должно быть вызвано ровно один раз")
		assert.Equal(t, markID, markRepo.deletedID, "удаляться должна запрошенная метка")
	})

	t.Run("ошибка получения метки пробрасывается", func(t *testing.T) {
		wantErr := errors.New("БД недоступна")
		markRepo := &fakeMarkRepo{getByIDErr: wantErr}
		s := newServiceWith(markRepo, nil, nil)

		err := s.DeleteMark(ctx, ownerID, markID)

		require.ErrorIs(t, err, wantErr)
		assert.Zero(t, markRepo.deleteCalls, "до удаления дело дойти не должно")
	})

	t.Run("чужую метку удалить нельзя", func(t *testing.T) {
		markRepo := &fakeMarkRepo{getByIDResult: liveMark(ownerID)}
		s := newServiceWith(markRepo, nil, nil)

		const strangerID = uint(999)
		err := s.DeleteMark(ctx, strangerID, markID)

		require.Error(t, err)
		assert.Zero(t, markRepo.deleteCalls, "чужая метка не должна удаляться")
	})

	t.Run("ошибка удаления пробрасывается", func(t *testing.T) {
		wantErr := errors.New("нарушение внешнего ключа")
		markRepo := &fakeMarkRepo{getByIDResult: liveMark(ownerID), deleteErr: wantErr}
		s := newServiceWith(markRepo, nil, nil)

		err := s.DeleteMark(ctx, ownerID, markID)

		require.ErrorIs(t, err, wantErr)
		assert.Equal(t, 1, markRepo.deleteCalls)
	})
}
