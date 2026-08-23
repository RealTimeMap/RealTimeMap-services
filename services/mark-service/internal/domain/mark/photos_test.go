package mark

import (
	"context"
	"errors"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тесты работы сервиса с фотографиями после перехода на content-addressed
// хранилище. Живой MinIO не нужен: store — интерфейс, подменяется фейком.
//
// Проверяется то, что изменилось при переезде:
//   - UploadMultiple стал fail-fast (раньше ошибки проглатывались)
//   - удаление фото из метки больше НЕ трогает байты в хранилище
//   - опции загрузки соответствуют горячему пути (Optimize отключён)

func newTestService(store storage.Storage) *Service {
	return newServiceWith(nil, nil, store)
}

func photoInput(name string) mediavalidator.PhotoInput {
	return mediavalidator.PhotoInput{
		Data:     []byte("байты " + name),
		FileName: name,
	}
}

func TestUploadPhotos(t *testing.T) {
	ctx := context.Background()

	t.Run("данные и опции доходят до хранилища без искажений", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		input := []mediavalidator.PhotoInput{photoInput("a.jpg"), photoInput("b.png")}

		photos, err := s.uploadPhotos(ctx, input)
		require.NoError(t, err)
		require.Len(t, photos, 2)

		require.Len(t, store.uploadMultipleCalls, 1, "должен быть ровно один пакетный вызов")
		files := store.uploadMultipleCalls[0]
		require.Len(t, files, 2)

		for i, want := range input {
			assert.Equal(t, want.Data, files[i].Data, "позиция %d: байты переданы как есть", i)
			assert.Equal(t, want.FileName, files[i].Options.FileName)
			assert.Equal(t, storage.CategoryMarkPhoto, files[i].Options.Category)
			assert.Equal(t, int64(5*1024*1024), files[i].Options.MaxSize)
			assert.False(t, files[i].Options.Optimize,
				"горячий путь: оптимизация отключена намеренно, до 10 фото за запрос")
		}
	})

	t.Run("порядок фотографий сохраняется", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		photos, err := s.uploadPhotos(ctx, []mediavalidator.PhotoInput{
			photoInput("первая.jpg"),
			photoInput("вторая.jpg"),
			photoInput("третья.jpg"),
		})
		require.NoError(t, err)
		require.Len(t, photos, 3)

		assert.Equal(t, "первая.jpg", photos[0].FileName)
		assert.Equal(t, "вторая.jpg", photos[1].FileName)
		assert.Equal(t, "третья.jpg", photos[2].FileName)
	})

	t.Run("ошибка хранилища возвращается наверх, а не проглатывается", func(t *testing.T) {
		// Ключевое изменение: раньше метка создавалась с неполным набором фото
		// и отвечала 200 OK. Теперь ошибка обязана дойти до вызывающего.
		wantErr := errors.New("бакет недоступен")
		store := &fakeStorage{err: wantErr}
		s := newTestService(store)

		photos, err := s.uploadPhotos(ctx, []mediavalidator.PhotoInput{photoInput("a.jpg")})
		require.Error(t, err)
		assert.Nil(t, photos, "частичный результат отдавать нельзя")
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("пустой список не ходит в хранилище зря", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		photos, err := s.uploadPhotos(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, photos)
	})
}

func TestUpdatePhotos(t *testing.T) {
	ctx := context.Background()

	existing := types.Photos{
		{URL: "https://example.test/files/v1/aa/bb/one.jpg", StorageKey: "v1/aa/bb/one.jpg"},
		{URL: "https://example.test/files/v1/cc/dd/two.jpg", StorageKey: "v1/cc/dd/two.jpg"},
		{URL: "https://example.test/files/v1/ee/ff/three.jpg", StorageKey: "v1/ee/ff/three.jpg"},
	}

	t.Run("удаление убирает ссылку, но НЕ трогает байты в хранилище", func(t *testing.T) {
		// Смысл проверки: при content-addressed ключах одно фото может
		// принадлежать двум меткам. Удаление объекта здесь стёрло бы файл и во
		// второй метке. Интерфейс Storage поэтому вообще не имеет Delete —
		// тест страхует от возврата такого вызова.
		store := &fakeStorage{}
		s := newTestService(store)

		result, err := s.updatePhotos(ctx, existing, nil, []string{existing[1].URL}, maxPhotosPerMark)
		require.NoError(t, err)

		require.Len(t, result, 2)
		assert.Equal(t, existing[0].URL, result[0].URL)
		assert.Equal(t, existing[2].URL, result[1].URL)

		assert.Empty(t, store.uploadMultipleCalls, "загрузки быть не должно")
		assert.Zero(t, store.totalCalls,
			"на пути удаления фото хранилище не должно трогаться вообще: "+
				"объект может принадлежать другой метке")
	})

	t.Run("удаление нескольких фото", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		result, err := s.updatePhotos(ctx, existing, nil,
			[]string{existing[0].URL, existing[2].URL}, maxPhotosPerMark)
		require.NoError(t, err)

		require.Len(t, result, 1)
		assert.Equal(t, existing[1].URL, result[0].URL)
	})

	t.Run("неизвестный URL в списке на удаление ничего не ломает", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		result, err := s.updatePhotos(ctx, existing, nil,
			[]string{"https://example.test/files/v1/zz/zz/нет-такого.jpg"}, maxPhotosPerMark)
		require.NoError(t, err)
		assert.Len(t, result, 3, "ни одно существующее фото не должно пропасть")
	})

	t.Run("оставшиеся и новые складываются в нужном порядке", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		result, err := s.updatePhotos(ctx, existing,
			[]mediavalidator.PhotoInput{photoInput("новая.jpg")},
			[]string{existing[0].URL},
			maxPhotosPerMark,
		)
		require.NoError(t, err)

		require.Len(t, result, 3)
		assert.Equal(t, existing[1].URL, result[0].URL)
		assert.Equal(t, existing[2].URL, result[1].URL)
		assert.Equal(t, "новая.jpg", result[2].FileName, "новые фото идут в конец")
	})

	t.Run("превышение лимита учитывает и старые, и новые", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		newOnes := make([]mediavalidator.PhotoInput, 3)
		for i := range newOnes {
			newOnes[i] = photoInput("новая.jpg")
		}

		// 3 старых + 3 новых = 6 при лимите 5.
		result, err := s.updatePhotos(ctx, existing, newOnes, nil, 5)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("maxPhotos=0 отключает проверку лимита", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		result, err := s.updatePhotos(ctx, existing,
			[]mediavalidator.PhotoInput{photoInput("новая.jpg")}, nil, 0)
		require.NoError(t, err)
		assert.Len(t, result, 4)
	})

	t.Run("ошибка загрузки заворачивается в доменную ошибку", func(t *testing.T) {
		wantErr := errors.New("бакет недоступен")
		store := &fakeStorage{err: wantErr}
		s := newTestService(store)

		result, err := s.updatePhotos(ctx, existing,
			[]mediavalidator.PhotoInput{photoInput("новая.jpg")}, nil, maxPhotosPerMark)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, wantErr, "исходная причина должна сохраниться в цепочке")
	})

	t.Run("удаление всех фото даёт пустой результат без ошибки", func(t *testing.T) {
		store := &fakeStorage{}
		s := newTestService(store)

		all := make([]string, len(existing))
		for i, p := range existing {
			all[i] = p.URL
		}

		result, err := s.updatePhotos(ctx, existing, nil, all, maxPhotosPerMark)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
