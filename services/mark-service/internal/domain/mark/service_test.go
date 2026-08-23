package mark

import (
	"context"
	"errors"
	"testing"
	"time"

	ctxHelper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тесты остальных методов сервисного слоя. Репозитории и хранилище —
// фейки из fakes_test.go, БД и MinIO не нужны.

func testUser() ctxHelper.UserInput {
	return ctxHelper.UserInput{UserID: 42, UserName: "tester"}
}

func validCreateParams() CreateMarkParams {
	return CreateMarkParams{
		MarkName:   "тестовая метка",
		CategoryId: 7,
		StartAt:    time.Now().UTC(),
		Geohash:    "ucfv0j",
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("метка собирается из входных данных и уходит в репозиторий", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		catRepo := &fakeCategoryRepo{getByIDResult: activeCategory()}
		s := newServiceWith(markRepo, catRepo, &fakeStorage{})

		input := validCreateParams()
		info := "подробности"
		input.AdditionalInfo = &info

		mark, err := s.Create(ctx, testUser(), input)
		require.NoError(t, err)
		require.NotNil(t, mark)

		require.Equal(t, 1, markRepo.createCalls)
		saved := markRepo.createdMark
		require.NotNil(t, saved)

		assert.Equal(t, input.MarkName, saved.MarkName)
		assert.Equal(t, &info, saved.AdditionalInfo)
		assert.Equal(t, input.CategoryId, saved.CategoryID)
		assert.Equal(t, input.Geohash, saved.Geohash)
		assert.Equal(t, uint(42), saved.UserID, "UserID берётся из контекста пользователя")
		assert.Equal(t, "tester", saved.UserName)
		assert.Equal(t, uint(7), catRepo.gotID, "категория проверяется по переданному id")
	})

	t.Run("без EndAt метка становится временной на час", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()}, &fakeStorage{})

		input := validCreateParams()
		input.EndAt = nil

		_, err := s.Create(ctx, testUser(), input)
		require.NoError(t, err)

		saved := markRepo.createdMark
		assert.True(t, saved.IsTemp, "метка без явного EndAt помечается временной")
		assert.Equal(t, input.StartAt.Add(time.Hour), saved.EndAt)
	})

	t.Run("явный EndAt сохраняется и метка не временная", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()}, &fakeStorage{})

		input := validCreateParams()
		endAt := input.StartAt.Add(5 * time.Hour)
		input.EndAt = &endAt

		_, err := s.Create(ctx, testUser(), input)
		require.NoError(t, err)

		saved := markRepo.createdMark
		assert.Equal(t, endAt, saved.EndAt)
		assert.False(t, saved.IsTemp)
	})

	t.Run("фото загружаются и попадают в метку", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		store := &fakeStorage{}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()}, store)

		input := validCreateParams()
		input.Photos = []mediavalidator.PhotoInput{photoInput("a.jpg"), photoInput("b.jpg")}

		_, err := s.Create(ctx, testUser(), input)
		require.NoError(t, err)

		require.Len(t, store.uploadMultipleCalls, 1)
		require.Len(t, markRepo.createdMark.Photos, 2)
		assert.Equal(t, "a.jpg", markRepo.createdMark.Photos[0].FileName)
	})

	t.Run("без фото в хранилище не ходим", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		store := &fakeStorage{}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()}, store)

		_, err := s.Create(ctx, testUser(), validCreateParams())
		require.NoError(t, err)

		assert.Zero(t, store.totalCalls)
		assert.Empty(t, markRepo.createdMark.Photos)
	})

	t.Run("ошибка загрузки фото отменяет создание метки", func(t *testing.T) {
		// Fail-fast: метка НЕ должна создаваться с неполным набором фото.
		markRepo := &fakeMarkRepo{}
		wantErr := errors.New("бакет недоступен")
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()},
			&fakeStorage{err: wantErr})

		input := validCreateParams()
		input.Photos = []mediavalidator.PhotoInput{photoInput("a.jpg")}

		mark, err := s.Create(ctx, testUser(), input)
		require.Error(t, err)
		assert.Nil(t, mark)
		assert.ErrorIs(t, err, wantErr)
		assert.Zero(t, markRepo.createCalls, "в БД ничего писать нельзя, раз фото не загрузились")
	})

	t.Run("неактивная категория отклоняется до записи", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		store := &fakeStorage{}
		s := newServiceWith(markRepo, &fakeCategoryRepo{
			getByIDResult: &category.Category{IsActive: false},
		}, store)

		input := validCreateParams()
		input.Photos = []mediavalidator.PhotoInput{photoInput("a.jpg")}

		mark, err := s.Create(ctx, testUser(), input)
		require.Error(t, err)
		assert.Nil(t, mark)
		assert.Zero(t, markRepo.createCalls)
		assert.Zero(t, store.totalCalls, "фото не должны заливаться при неверной категории")
	})

	t.Run("несуществующая категория пробрасывает ошибку репозитория", func(t *testing.T) {
		wantErr := errors.New("категория не найдена")
		markRepo := &fakeMarkRepo{}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDErr: wantErr}, &fakeStorage{})

		_, err := s.Create(ctx, testUser(), validCreateParams())
		require.ErrorIs(t, err, wantErr)
		assert.Zero(t, markRepo.createCalls)
	})

	t.Run("превышение дневного лимита отклоняет создание", func(t *testing.T) {
		markRepo := &fakeMarkRepo{todayCount: maxMarksPerDay + 1}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()}, &fakeStorage{})

		_, err := s.Create(ctx, testUser(), validCreateParams())
		require.Error(t, err)
		assert.Zero(t, markRepo.createCalls)
	})

	t.Run("StartAt слишком далеко в прошлом", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()}, &fakeStorage{})

		input := validCreateParams()
		input.StartAt = time.Now().UTC().AddDate(0, 0, -(maxStartAtPastDays + 1))

		_, err := s.Create(ctx, testUser(), input)
		require.Error(t, err)
		assert.Zero(t, markRepo.createCalls)
	})

	t.Run("StartAt слишком далеко в будущем", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()}, &fakeStorage{})

		input := validCreateParams()
		input.StartAt = time.Now().UTC().AddDate(0, 0, maxStartAtFutureDays+1)

		_, err := s.Create(ctx, testUser(), input)
		require.Error(t, err)
		assert.Zero(t, markRepo.createCalls)
	})

	t.Run("ошибка репозитория при создании пробрасывается", func(t *testing.T) {
		wantErr := errors.New("БД недоступна")
		markRepo := &fakeMarkRepo{createErr: wantErr}
		s := newServiceWith(markRepo, &fakeCategoryRepo{getByIDResult: activeCategory()}, &fakeStorage{})

		mark, err := s.Create(ctx, testUser(), validCreateParams())
		require.ErrorIs(t, err, wantErr)
		assert.Nil(t, mark)
	})
}

func TestUpdateMark(t *testing.T) {
	ctx := context.Background()
	const ownerID, markID = uint(42), uint(100)

	existingMark := func() *Mark {
		info := "старое описание"
		return &Mark{
			MarkName:       "старое имя",
			AdditionalInfo: &info,
			UserID:         ownerID,
			Photos: types.Photos{
				{URL: "https://example.test/files/v1/aa/bb/one.jpg", StorageKey: "v1/aa/bb/one.jpg"},
			},
		}
	}

	t.Run("обновляются только переданные поля", func(t *testing.T) {
		markRepo := &fakeMarkRepo{getByIDResult: existingMark()}
		s := newServiceWith(markRepo, nil, &fakeStorage{})

		_, err := s.UpdateMark(ctx, UpdateMarkParams{MarkName: "новое имя"}, ownerID, markID)
		require.NoError(t, err)

		require.Equal(t, 1, markRepo.updateCalls)
		assert.Equal(t, markID, markRepo.updatedID)
		assert.Equal(t, "новое имя", markRepo.updatedMark.MarkName)
		assert.Equal(t, "старое описание", *markRepo.updatedMark.AdditionalInfo,
			"непереданное поле не должно затираться")
	})

	t.Run("пустое имя не затирает существующее", func(t *testing.T) {
		markRepo := &fakeMarkRepo{getByIDResult: existingMark()}
		s := newServiceWith(markRepo, nil, &fakeStorage{})

		_, err := s.UpdateMark(ctx, UpdateMarkParams{MarkName: ""}, ownerID, markID)
		require.NoError(t, err)
		assert.Equal(t, "старое имя", markRepo.updatedMark.MarkName)
	})

	t.Run("чужую метку править нельзя", func(t *testing.T) {
		markRepo := &fakeMarkRepo{getByIDResult: existingMark()}
		store := &fakeStorage{}
		s := newServiceWith(markRepo, nil, store)

		const strangerID = uint(999)
		mark, err := s.UpdateMark(ctx, UpdateMarkParams{MarkName: "взлом"}, strangerID, markID)
		require.Error(t, err)
		assert.Nil(t, mark)
		assert.Zero(t, markRepo.updateCalls, "запись в БД не должна произойти")
		assert.Zero(t, store.totalCalls, "и в хранилище тоже не ходим")
	})

	t.Run("новые фото добавляются к существующим", func(t *testing.T) {
		markRepo := &fakeMarkRepo{getByIDResult: existingMark()}
		s := newServiceWith(markRepo, nil, &fakeStorage{})

		_, err := s.UpdateMark(ctx, UpdateMarkParams{
			Photos: []mediavalidator.PhotoInput{photoInput("новая.jpg")},
		}, ownerID, markID)
		require.NoError(t, err)

		require.Len(t, markRepo.updatedMark.Photos, 2)
		assert.Equal(t, "новая.jpg", markRepo.updatedMark.Photos[1].FileName)
	})

	t.Run("удаление фото не трогает байты в хранилище", func(t *testing.T) {
		mark := existingMark()
		markRepo := &fakeMarkRepo{getByIDResult: mark}
		store := &fakeStorage{}
		s := newServiceWith(markRepo, nil, store)

		_, err := s.UpdateMark(ctx, UpdateMarkParams{
			PhotosToDelete: []string{mark.Photos[0].URL},
		}, ownerID, markID)
		require.NoError(t, err)

		assert.Empty(t, markRepo.updatedMark.Photos)
		assert.Zero(t, store.totalCalls,
			"объект может принадлежать другой метке — удалять его нельзя")
	})

	t.Run("метка не найдена", func(t *testing.T) {
		wantErr := errors.New("метка не найдена")
		markRepo := &fakeMarkRepo{getByIDErr: wantErr}
		s := newServiceWith(markRepo, nil, &fakeStorage{})

		mark, err := s.UpdateMark(ctx, UpdateMarkParams{MarkName: "x"}, ownerID, markID)
		require.ErrorIs(t, err, wantErr)
		assert.Nil(t, mark)
		assert.Zero(t, markRepo.updateCalls)
	})

	t.Run("ошибка загрузки отменяет обновление", func(t *testing.T) {
		wantErr := errors.New("бакет недоступен")
		markRepo := &fakeMarkRepo{getByIDResult: existingMark()}
		s := newServiceWith(markRepo, nil, &fakeStorage{err: wantErr})

		mark, err := s.UpdateMark(ctx, UpdateMarkParams{
			Photos: []mediavalidator.PhotoInput{photoInput("новая.jpg")},
		}, ownerID, markID)
		require.Error(t, err)
		assert.Nil(t, mark)
		assert.Zero(t, markRepo.updateCalls, "в БД писать нельзя, раз фото не загрузились")
	})

	t.Run("ошибка репозитория при сохранении пробрасывается", func(t *testing.T) {
		wantErr := errors.New("БД недоступна")
		markRepo := &fakeMarkRepo{getByIDResult: existingMark(), updateErr: wantErr}
		s := newServiceWith(markRepo, nil, &fakeStorage{})

		mark, err := s.UpdateMark(ctx, UpdateMarkParams{MarkName: "новое"}, ownerID, markID)
		require.ErrorIs(t, err, wantErr)
		assert.Nil(t, mark)
	})
}

func TestApplyUpdates(t *testing.T) {
	s := newServiceWith(nil, nil, nil)

	t.Run("непустое имя перезаписывает", func(t *testing.T) {
		m := &Mark{MarkName: "старое"}
		s.applyUpdates(m, UpdateMarkParams{MarkName: "новое"})
		assert.Equal(t, "новое", m.MarkName)
	})

	t.Run("пустое имя игнорируется", func(t *testing.T) {
		m := &Mark{MarkName: "старое"}
		s.applyUpdates(m, UpdateMarkParams{MarkName: ""})
		assert.Equal(t, "старое", m.MarkName)
	})

	t.Run("nil AdditionalInfo игнорируется, не-nil перезаписывает", func(t *testing.T) {
		old := "старое"
		m := &Mark{AdditionalInfo: &old}

		s.applyUpdates(m, UpdateMarkParams{})
		assert.Equal(t, &old, m.AdditionalInfo)

		empty := ""
		s.applyUpdates(m, UpdateMarkParams{AdditionalInfo: &empty})
		assert.Equal(t, "", *m.AdditionalInfo,
			"явно переданная пустая строка — это осознанная очистка поля")
	})
}

func TestCheckOwnerShip(t *testing.T) {
	s := newServiceWith(nil, nil, nil)

	t.Run("владелец проходит", func(t *testing.T) {
		assert.NoError(t, s.checkOwnerShip(&Mark{UserID: 42}, 42))
	})

	t.Run("чужой пользователь отклоняется", func(t *testing.T) {
		assert.Error(t, s.checkOwnerShip(&Mark{UserID: 42}, 43))
	})
}

func TestGetMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("GetMarksInArea передаёт фильтр и возвращает результат", func(t *testing.T) {
		want := []*Mark{{MarkName: "первая"}, {MarkName: "вторая"}}
		markRepo := &fakeMarkRepo{areaResult: want}
		s := newServiceWith(markRepo, nil, nil)

		filter := Filter{ZoomLevel: 12, Duration: 3}
		got, err := s.GetMarksInArea(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, filter, markRepo.gotFilter)
	})

	t.Run("GetMarksInArea пробрасывает ошибку", func(t *testing.T) {
		wantErr := errors.New("БД недоступна")
		s := newServiceWith(&fakeMarkRepo{areaErr: wantErr}, nil, nil)

		got, err := s.GetMarksInArea(ctx, Filter{})
		require.ErrorIs(t, err, wantErr)
		assert.Nil(t, got)
	})

	t.Run("GetMarksInCluster передаёт фильтр и возвращает результат", func(t *testing.T) {
		want := []*Cluster{{Count: 5}}
		markRepo := &fakeMarkRepo{clusterResult: want}
		s := newServiceWith(markRepo, nil, nil)

		filter := Filter{ZoomLevel: 5, Duration: 1}
		got, err := s.GetMarksInCluster(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, filter, markRepo.gotFilter)
	})

	t.Run("GetMarksInCluster пробрасывает ошибку", func(t *testing.T) {
		wantErr := errors.New("БД недоступна")
		s := newServiceWith(&fakeMarkRepo{clusterErr: wantErr}, nil, nil)

		got, err := s.GetMarksInCluster(ctx, Filter{})
		require.ErrorIs(t, err, wantErr)
		assert.Nil(t, got)
	})

	t.Run("GetDetailMark возвращает метку", func(t *testing.T) {
		want := &Mark{MarkName: "искомая"}
		s := newServiceWith(&fakeMarkRepo{getByIDResult: want}, nil, nil)

		got, err := s.GetDetailMark(ctx, 100)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("GetDetailMark пробрасывает ошибку", func(t *testing.T) {
		wantErr := errors.New("не найдено")
		s := newServiceWith(&fakeMarkRepo{getByIDErr: wantErr}, nil, nil)

		got, err := s.GetDetailMark(ctx, 100)
		require.ErrorIs(t, err, wantErr)
		assert.Nil(t, got)
	})
}

func TestGetUserMarks(t *testing.T) {
	ctx := context.Background()

	t.Run("возвращает метки и общее количество", func(t *testing.T) {
		want := []*Mark{{MarkName: "моя"}}
		markRepo := &fakeMarkRepo{userMarks: want, userMarksCount: 17}
		s := newServiceWith(markRepo, nil, nil)

		got, count, err := s.GetUserMarks(ctx, 42, pagination.Params{Page: 2, PageSize: 10})
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(17), count)
		assert.Equal(t, uint(42), markRepo.gotUserID)
	})

	t.Run("пустая пагинация дополняется значениями по умолчанию", func(t *testing.T) {
		markRepo := &fakeMarkRepo{}
		s := newServiceWith(markRepo, nil, nil)

		_, _, err := s.GetUserMarks(ctx, 42, pagination.Params{})
		require.NoError(t, err)

		assert.NotZero(t, markRepo.gotPagination.PageSize,
			"Defaults() должен проставить размер страницы, иначе репозиторий получит 0")
		assert.NotZero(t, markRepo.gotPagination.Page)
	})

	t.Run("пробрасывает ошибку и не возвращает счётчик", func(t *testing.T) {
		wantErr := errors.New("БД недоступна")
		s := newServiceWith(&fakeMarkRepo{userMarksErr: wantErr}, nil, nil)

		got, count, err := s.GetUserMarks(ctx, 42, pagination.Params{})
		require.ErrorIs(t, err, wantErr)
		assert.Nil(t, got)
		assert.Zero(t, count)
	})
}
