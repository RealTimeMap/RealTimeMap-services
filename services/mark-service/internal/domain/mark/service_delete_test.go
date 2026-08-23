package mark

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ВНИМАНИЕ. Тесты ниже фиксируют ФАКТИЧЕСКОЕ поведение DeleteMark, а оно
// сломано. Они намеренно не выражают желаемое поведение — иначе пакет был бы
// красным, — но каждый помечен комментарием с описанием дефекта.
//
// Два дефекта, оба видны прямо в теле метода:
//
//  1. Метод НИЧЕГО НЕ УДАЛЯЕТ. markRepo.Delete не вызывается ни разу, метод
//     просто возвращает nil. При этом он подключён к живому маршруту
//     DELETE /api/v2/marks/:markID (mark_handler.go:40), то есть клиент
//     получает успешный ответ, а метка остаётся в БД.
//
//  2. Условие `if !obj.DeletedAt.Valid` инвертировано. DeletedAt.Valid=true
//     означает «запись уже удалена» (soft delete в gorm). Сейчас метод
//     отвечает «не найдено» для ЖИВОЙ метки и пропускает дальше уже удалённую.
//
// Когда метод будут чинить, эти тесты обязаны упасть — это и есть сигнал, что
// пора переписать их под правильное поведение.

func TestDeleteMark(t *testing.T) {
	ctx := context.Background()
	const ownerID, markID = uint(42), uint(100)

	// deletedMark — метка, помеченная как удалённая (DeletedAt проставлен).
	deletedMark := func(userID uint) *Mark {
		m := &Mark{UserID: userID}
		m.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
		return m
	}

	// liveMark — обычная живая метка (DeletedAt не проставлен).
	liveMark := func(userID uint) *Mark {
		return &Mark{UserID: userID}
	}

	t.Run("ошибка получения метки пробрасывается", func(t *testing.T) {
		wantErr := errors.New("БД недоступна")
		s := newServiceWith(&fakeMarkRepo{getByIDErr: wantErr}, nil, nil)

		err := s.DeleteMark(ctx, ownerID, markID)
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("ДЕФЕКТ: живая метка отвергается как ненайденная", func(t *testing.T) {
		// Правильное поведение: живую метку надо удалять.
		// Фактическое: возвращается ErrMarkNotFound из-за инвертированного
		// условия `if !obj.DeletedAt.Valid`.
		markRepo := &fakeMarkRepo{getByIDResult: liveMark(ownerID)}
		s := newServiceWith(markRepo, nil, nil)

		err := s.DeleteMark(ctx, ownerID, markID)
		require.Error(t, err, "фиксируем текущее поведение: живую метку удалить нельзя")
		assert.Zero(t, markRepo.deleteCalls)
	})

	t.Run("ДЕФЕКТ: чужая уже удалённая метка отвергается по правам", func(t *testing.T) {
		// Здесь проверка прав отрабатывает верно — на неё дефект не влияет.
		markRepo := &fakeMarkRepo{getByIDResult: deletedMark(ownerID)}
		s := newServiceWith(markRepo, nil, nil)

		const strangerID = uint(999)
		err := s.DeleteMark(ctx, strangerID, markID)
		require.Error(t, err)
		assert.Zero(t, markRepo.deleteCalls)
	})

	t.Run("ДЕФЕКТ: успешный путь не удаляет метку", func(t *testing.T) {
		// Владелец + уже удалённая метка — единственная комбинация, дающая nil.
		// И даже на ней markRepo.Delete не вызывается: метод возвращает успех,
		// ничего не сделав. Именно это и получает клиент по DELETE-маршруту.
		markRepo := &fakeMarkRepo{getByIDResult: deletedMark(ownerID)}
		s := newServiceWith(markRepo, nil, nil)

		err := s.DeleteMark(ctx, ownerID, markID)
		require.NoError(t, err)
		assert.Zero(t, markRepo.deleteCalls,
			"метод возвращает успех, но удаления не происходит — см. комментарий к файлу")
	})
}
