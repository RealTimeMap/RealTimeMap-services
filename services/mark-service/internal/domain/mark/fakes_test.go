package mark

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/mark/category"
)

// Общие фейки для тестов сервисного слоя. Пишем руками, а не генерируем:
// проверять надо переданные аргументы и факт/количество вызовов, и этого
// достаточно. Все поля — публичные для теста, настраиваются на месте.

// fakeStorage — реализация storage.Storage под тесты.
type fakeStorage struct {
	// uploadMultipleCalls — все наборы файлов, с которыми звали UploadMultiple.
	uploadMultipleCalls [][]storage.FileUpload
	// err — если задана, UploadMultiple возвращает её (fail-fast).
	err error
	// uploadCalls — счётчик одиночных Upload (сервис их не использует).
	uploadCalls int
	// totalCalls — счётчик ЛЮБЫХ обращений к хранилищу. Нужен, чтобы поймать
	// возврат удаления файлов: интерфейс больше не имеет Delete, но попытку
	// сходить в хранилище на пути удаления фото (хоть Exists, хоть GetURL)
	// тест обязан заметить.
	totalCalls int
}

func (f *fakeStorage) Upload(_ context.Context, _ []byte, _ storage.UploadOptions) (*types.Photo, error) {
	f.uploadCalls++
	f.totalCalls++
	return &types.Photo{}, nil
}

func (f *fakeStorage) UploadMultiple(_ context.Context, files []storage.FileUpload) (types.Photos, error) {
	f.uploadMultipleCalls = append(f.uploadMultipleCalls, files)
	f.totalCalls++
	if f.err != nil {
		return nil, f.err
	}

	photos := make(types.Photos, len(files))
	for i, file := range files {
		key := "v1/ab/cd/hash" + file.Options.FileName
		photos[i] = types.Photo{
			URL:        "https://example.test/files/" + key,
			StorageKey: key,
			FileName:   file.Options.FileName,
		}
	}
	return photos, nil
}

func (f *fakeStorage) GetURL(storageKey string) string {
	f.totalCalls++
	return "https://example.test/files/" + storageKey
}

func (f *fakeStorage) Exists(_ context.Context, _ string) (bool, error) {
	f.totalCalls++
	return true, nil
}

// fakeMarkRepo — реализация Repository под тесты.
type fakeMarkRepo struct {
	// Возвращаемые значения.
	createResult   *Mark
	createErr      error
	getByIDResult  *Mark
	getByIDErr     error
	updateResult   *Mark
	updateErr      error
	todayCount     int64
	todayErr       error
	areaResult     []*Mark
	areaErr        error
	clusterResult  []*Cluster
	clusterErr     error
	userMarks      []*Mark
	userMarksCount int64
	userMarksErr   error

	// Записанные аргументы вызовов.
	createdMark   *Mark
	updatedMark   *Mark
	updatedID     uint
	gotFilter     Filter
	gotUserID     uint
	gotPagination pagination.Params

	createCalls int
	updateCalls int
	deleteCalls int
}

func (f *fakeMarkRepo) Create(_ context.Context, data *Mark) (*Mark, error) {
	f.createCalls++
	f.createdMark = data
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResult != nil {
		return f.createResult, nil
	}
	return data, nil
}

func (f *fakeMarkRepo) TodayCreated(_ context.Context, _ uint) (int64, error) {
	return f.todayCount, f.todayErr
}

func (f *fakeMarkRepo) GetMarksInArea(_ context.Context, filter Filter) ([]*Mark, error) {
	f.gotFilter = filter
	return f.areaResult, f.areaErr
}

func (f *fakeMarkRepo) GetUserMarks(_ context.Context, userID uint, params pagination.Params) ([]*Mark, int64, error) {
	f.gotUserID = userID
	f.gotPagination = params
	if f.userMarksErr != nil {
		return nil, 0, f.userMarksErr
	}
	return f.userMarks, f.userMarksCount, nil
}

func (f *fakeMarkRepo) GetMarksInCluster(_ context.Context, filter Filter) ([]*Cluster, error) {
	f.gotFilter = filter
	return f.clusterResult, f.clusterErr
}

func (f *fakeMarkRepo) Exist(_ context.Context, _ uint) (bool, error) {
	return true, nil
}

func (f *fakeMarkRepo) Delete(_ context.Context, _ uint) error {
	f.deleteCalls++
	return nil
}

func (f *fakeMarkRepo) GetByID(_ context.Context, _ uint) (*Mark, error) {
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	return f.getByIDResult, nil
}

func (f *fakeMarkRepo) Update(_ context.Context, id uint, mark *Mark) (*Mark, error) {
	f.updateCalls++
	f.updatedID = id
	f.updatedMark = mark
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.updateResult != nil {
		return f.updateResult, nil
	}
	return mark, nil
}

func (f *fakeMarkRepo) IncShare(_ context.Context, _ uint) (int64, error) {
	return 0, nil
}

// fakeCategoryRepo — реализация category.Repository под тесты.
type fakeCategoryRepo struct {
	getByIDResult *category.Category
	getByIDErr    error
	gotID         uint
}

func (f *fakeCategoryRepo) Create(_ context.Context, data *category.Category) (*category.Category, error) {
	return data, nil
}

func (f *fakeCategoryRepo) GetByName(_ context.Context, _ string) (*category.Category, error) {
	return f.getByIDResult, f.getByIDErr
}

func (f *fakeCategoryRepo) GetByID(_ context.Context, id uint) (*category.Category, error) {
	f.gotID = id
	if f.getByIDErr != nil {
		return nil, f.getByIDErr
	}
	return f.getByIDResult, nil
}

func (f *fakeCategoryRepo) Exist(_ context.Context, _ int) (bool, error) {
	return true, nil
}

func (f *fakeCategoryRepo) GetAll(_ context.Context) ([]*category.Category, error) {
	return nil, nil
}

// Фейки обязаны удовлетворять интерфейсам — иначе тесты молча разойдутся с кодом.
var (
	_ storage.Storage     = (*fakeStorage)(nil)
	_ Repository          = (*fakeMarkRepo)(nil)
	_ category.Repository = (*fakeCategoryRepo)(nil)
)

// activeCategory — категория, проходящая валидацию.
func activeCategory() *category.Category {
	return &category.Category{IsActive: true}
}

// newServiceWith собирает сервис со всеми подменёнными зависимостями.
func newServiceWith(markRepo Repository, catRepo category.Repository, store storage.Storage) *Service {
	return NewService(markRepo, catRepo, store, logger.NewNop())
}
