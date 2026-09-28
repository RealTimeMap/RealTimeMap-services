// Package asyncstorage — ВРЕМЕННАЯ обёртка над storage.Storage, которая
// выносит запись фото в S3 из HTTP-запроса в фоновый пул воркеров.
//
// Работает за счёт content-addressed ключей: URL объекта выводится из хеша
// байтов (storage.Resolve), поэтому его можно отдать клиенту и сохранить в
// метку до того, как объект реально появится в бакете.
//
// Цена решения:
//   - пока воркер не записал объект, URL отдаёт 404 (обычно секунды);
//   - если запись не удалась после всех попыток, метка остаётся со ссылкой
//     на несуществующий объект — такие случаи логируются на уровне Error
//     с ключом, повторно их никто не заливает;
//   - при падении процесса (не graceful shutdown) очередь теряется.
package asyncstorage

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/imageprocessor"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

const (
	defaultWorkers   = 4
	defaultQueueSize = 50 // по одному файлу; при лимите 5 МБ — до ~250 МБ в памяти
	defaultAttempts  = 3
	defaultBackoff   = time.Second
	uploadTimeout    = 30 * time.Second
)

type Config struct {
	Workers   int
	QueueSize int
	Attempts  int
	Backoff   time.Duration
}

type job struct {
	data []byte
	opts storage.UploadOptions
	key  string
}

// Storage реализует storage.Storage и runner.Server.
type Storage struct {
	inner     storage.Storage
	processor *imageprocessor.Processor
	logger    *zap.Logger
	cfg       Config

	queue chan job

	mu      sync.RWMutex
	closed  bool
	wg      sync.WaitGroup
	stopped chan struct{}
}

var _ storage.Storage = (*Storage)(nil)

// New запускает воркеры сразу: Upload может прийти раньше, чем runner вызовет Run.
func New(inner storage.Storage, cfg Config, logger *zap.Logger) *Storage {
	if cfg.Workers <= 0 {
		cfg.Workers = defaultWorkers
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultQueueSize
	}
	if cfg.Attempts <= 0 {
		cfg.Attempts = defaultAttempts
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = defaultBackoff
	}

	s := &Storage{
		inner:     inner,
		processor: imageprocessor.NewProcessor(logger),
		logger:    logger.With(zap.String("component", "asyncstorage")),
		cfg:       cfg,
		queue:     make(chan job, cfg.QueueSize),
		stopped:   make(chan struct{}),
	}

	s.wg.Add(cfg.Workers)
	for range cfg.Workers {
		go s.worker()
	}
	return s
}

// Upload валидирует файл и считает его метаданные синхронно, а запись в бакет
// ставит в очередь. Если очередь заполнена или пул остановлен, загружает
// синхронно — лучше медленный ответ, чем потерянное фото.
func (s *Storage) Upload(ctx context.Context, data []byte, opts storage.UploadOptions) (*types.Photo, error) {
	obj, err := storage.Resolve(data, opts)
	if err != nil {
		return nil, err
	}

	if !s.enqueue(job{data: data, opts: opts, key: obj.Key}) {
		s.logger.Warn("upload queue unavailable, uploading synchronously", zap.String("key", obj.Key))
		return s.inner.Upload(ctx, data, opts)
	}

	width, height := s.processor.GetDimensionsFast(data)
	return &types.Photo{
		URL:        s.inner.GetURL(obj.Key),
		FileName:   opts.FileName,
		Size:       int64(len(data)), // при Optimize: true реальный размер в бакете будет меньше
		Width:      width,
		Height:     height,
		MimeType:   obj.MimeType,
		Hash:       obj.Hash,
		StorageKey: obj.Key,
		UploadedAt: time.Now(),
	}, nil
}

// UploadMultiple сохраняет fail-fast контракт для того, что можно проверить
// синхронно (валидация файлов). Ошибки самой записи в S3 после ответа клиенту
// вернуть уже нельзя — см. комментарий пакета.
func (s *Storage) UploadMultiple(ctx context.Context, files []storage.FileUpload) (types.Photos, error) {
	// Сначала валидируем всё, чтобы невалидный файл не оставил в очереди
	// «осиротевшие» загрузки остальных.
	for i, f := range files {
		if _, err := storage.Resolve(f.Data, f.Options); err != nil {
			return nil, fmt.Errorf("file %d (%s): %w", i, f.Options.FileName, err)
		}
	}

	photos := make(types.Photos, 0, len(files))
	for i, f := range files {
		photo, err := s.Upload(ctx, f.Data, f.Options)
		if err != nil {
			return nil, fmt.Errorf("file %d (%s): %w", i, f.Options.FileName, err)
		}
		photos = append(photos, *photo)
	}
	return photos, nil
}

func (s *Storage) GetURL(storageKey string) string {
	return s.inner.GetURL(storageKey)
}

func (s *Storage) Exists(ctx context.Context, storageKey string) (bool, error) {
	return s.inner.Exists(ctx, storageKey)
}

func (s *Storage) enqueue(j job) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return false
	}
	select {
	case s.queue <- j:
		return true
	default:
		return false
	}
}

func (s *Storage) worker() {
	defer s.wg.Done()
	for j := range s.queue {
		s.process(j)
	}
}

func (s *Storage) process(j job) {
	var err error
	for attempt := 1; attempt <= s.cfg.Attempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), uploadTimeout)
		_, err = s.inner.Upload(ctx, j.data, j.opts)
		cancel()
		if err == nil {
			s.logger.Debug("photo uploaded", zap.String("key", j.key), zap.Int("attempt", attempt))
			return
		}
		// Валидационные ошибки не лечатся повтором.
		if errors.Is(err, storage.ErrInvalidMimeType) || errors.Is(err, storage.ErrFileTooLarge) ||
			errors.Is(err, storage.ErrInvalidCategory) {
			break
		}
		if attempt < s.cfg.Attempts {
			s.logger.Warn("photo upload failed, retrying",
				zap.String("key", j.key), zap.Int("attempt", attempt), zap.Error(err))
			time.Sleep(s.cfg.Backoff * time.Duration(attempt))
		}
	}
	s.logger.Error("photo upload failed permanently, mark references missing object",
		zap.String("key", j.key), zap.String("file", j.opts.FileName), zap.Error(err))
}

// Run нужен для runner.Server: воркеры уже запущены в New, здесь только ждём остановки.
func (s *Storage) Run() error {
	<-s.stopped
	return nil
}

// Shutdown перестаёт принимать задачи (новые Upload пойдут синхронно) и ждёт,
// пока воркеры дольют очередь, но не дольше дедлайна ctx.
func (s *Storage) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	pending := len(s.queue)
	close(s.queue)
	s.mu.Unlock()

	s.logger.Info("draining upload queue", zap.Int("pending", pending))

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	defer close(s.stopped)
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.logger.Error("upload queue not drained before shutdown deadline", zap.Int("left", len(s.queue)))
		return ctx.Err()
	}
}
