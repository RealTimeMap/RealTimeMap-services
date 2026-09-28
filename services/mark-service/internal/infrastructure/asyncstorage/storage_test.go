package asyncstorage

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

// fakeStorage имитирует MinIOStorage: ключ считает через storage.Resolve,
// как настоящий Upload.
type fakeStorage struct {
	mu       sync.Mutex
	uploaded map[string]bool
	calls    atomic.Int32
	failures atomic.Int32 // сколько первых вызовов завершить ошибкой
	block    chan struct{}
}

func newFake() *fakeStorage { return &fakeStorage{uploaded: map[string]bool{}} }

func (f *fakeStorage) Upload(_ context.Context, data []byte, opts storage.UploadOptions) (*types.Photo, error) {
	if f.block != nil {
		<-f.block
	}
	f.calls.Add(1)
	if f.failures.Add(-1) >= 0 {
		return nil, errors.New("s3 unavailable")
	}
	obj, err := storage.Resolve(data, opts)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.uploaded[obj.Key] = true
	f.mu.Unlock()
	return &types.Photo{URL: f.GetURL(obj.Key), StorageKey: obj.Key}, nil
}

func (f *fakeStorage) UploadMultiple(context.Context, []storage.FileUpload) (types.Photos, error) {
	panic("not used")
}
func (f *fakeStorage) GetURL(key string) string { return "https://cdn.test/" + key }
func (f *fakeStorage) Exists(_ context.Context, key string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.uploaded[key], nil
}

func pngBytes(t *testing.T, seed uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Pix[0] = seed
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func file(t *testing.T, seed uint8) storage.FileUpload {
	return storage.FileUpload{
		Data:    pngBytes(t, seed),
		Options: storage.UploadOptions{FileName: "a.png", Category: storage.CategoryMarkPhoto},
	}
}

func TestUploadMultiple_ReturnsFinalURLsAndUploadsInBackground(t *testing.T) {
	inner := newFake()
	s := New(inner, Config{Backoff: time.Millisecond}, zap.NewNop())

	photos, err := s.UploadMultiple(context.Background(), []storage.FileUpload{file(t, 1), file(t, 2)})
	require.NoError(t, err)
	require.Len(t, photos, 2)
	assert.Equal(t, 4, photos[0].Width)
	assert.Equal(t, 3, photos[0].Height)

	require.NoError(t, s.Shutdown(context.Background()))

	for _, p := range photos {
		ok, _ := inner.Exists(context.Background(), p.StorageKey)
		assert.True(t, ok, "объект %s должен быть загружен", p.StorageKey)
		assert.Equal(t, inner.GetURL(p.StorageKey), p.URL)
	}
}

func TestUploadMultiple_InvalidFileEnqueuesNothing(t *testing.T) {
	inner := newFake()
	s := New(inner, Config{}, zap.NewNop())

	bad := storage.FileUpload{Data: []byte("not an image"), Options: storage.UploadOptions{Category: storage.CategoryMarkPhoto}}
	_, err := s.UploadMultiple(context.Background(), []storage.FileUpload{file(t, 1), bad})
	require.ErrorIs(t, err, storage.ErrInvalidMimeType)

	require.NoError(t, s.Shutdown(context.Background()))
	assert.Zero(t, inner.calls.Load())
}

func TestWorker_RetriesTransientErrors(t *testing.T) {
	inner := newFake()
	inner.failures.Store(2)
	s := New(inner, Config{Attempts: 3, Backoff: time.Millisecond}, zap.NewNop())

	photo, err := s.Upload(context.Background(), file(t, 1).Data, file(t, 1).Options)
	require.NoError(t, err)
	require.NoError(t, s.Shutdown(context.Background()))

	ok, _ := inner.Exists(context.Background(), photo.StorageKey)
	assert.True(t, ok)
	assert.EqualValues(t, 3, inner.calls.Load())
}

func TestUpload_FallsBackToSyncWhenQueueFullOrClosed(t *testing.T) {
	inner := newFake()
	inner.block = make(chan struct{})
	s := New(inner, Config{Workers: 1, QueueSize: 1}, zap.NewNop())

	// Первый файл забирает воркер (и висит на block), второй занимает очередь.
	_, err := s.Upload(context.Background(), file(t, 1).Data, file(t, 1).Options)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(s.queue) == 0 }, time.Second, time.Millisecond)
	_, err = s.Upload(context.Background(), file(t, 2).Data, file(t, 2).Options)
	require.NoError(t, err)

	// Третий не влезает — синхронная загрузка, ждёт того же block.
	done := make(chan struct{})
	go func() {
		_, err := s.Upload(context.Background(), file(t, 3).Data, file(t, 3).Options)
		assert.NoError(t, err)
		close(done)
	}()
	close(inner.block)
	<-done

	require.NoError(t, s.Shutdown(context.Background()))
	assert.EqualValues(t, 3, inner.calls.Load())

	// После остановки — тоже синхронно, без паники на закрытом канале.
	_, err = s.Upload(context.Background(), file(t, 4).Data, file(t, 4).Options)
	require.NoError(t, err)
	assert.EqualValues(t, 4, inner.calls.Load())
}
