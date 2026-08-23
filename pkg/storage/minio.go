package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/imageprocessor"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// maxWorkers ограничивает число параллельных загрузок внутри UploadMultiple.
const maxWorkers = 5

// MinIOStorage — реализация Storage поверх S3-совместимого хранилища.
type MinIOStorage struct {
	client    *minio.Client
	bucket    string
	baseURL   string
	processor *imageprocessor.Processor
	logger    *zap.Logger
}

// NewMinIOStorage создаёт клиент и проверяет, что бакет существует.
//
// Бакет намеренно НЕ создаётся из кода: политику публичного чтения префикса
// v1/ выставляет minio-init в docker-compose. Созданный отсюда бакет был бы
// приватным, и раздача через traefik молча сломалась бы.
func NewMinIOStorage(cfg StorageConfig, logger *zap.Logger) (Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket %q: %w", cfg.Bucket, err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket %q does not exist: создаётся сервисом minio-init", cfg.Bucket)
	}

	return &MinIOStorage{
		client:    client,
		bucket:    cfg.Bucket,
		baseURL:   cfg.BaseURL,
		processor: imageprocessor.NewProcessor(logger),
		logger:    logger,
	}, nil
}

// buildKey строит content-addressed ключ: v1/ab/cd/<sha256><ext>.
//
// Шардирование по первым 4 символам хеша — чтобы не складывать миллионы
// объектов в один префикс. Префикс v1/ позволяет инвалидировать всё разом при
// смене параметров оптимизации: ключ считается от исходника и сам по себе
// никогда не изменится.
func buildKey(hash string, ext string) string {
	return fmt.Sprintf("v1/%s/%s/%s%s", hash[0:2], hash[2:4], hash, ext)
}

// Upload загружает файл, если объекта с таким содержимым ещё нет.
func (s *MinIOStorage) Upload(ctx context.Context, data []byte, opts UploadOptions) (*types.Photo, error) {
	if err := opts.Category.Validate(); err != nil {
		return nil, err
	}

	if opts.MaxSize > 0 && int64(len(data)) > opts.MaxSize {
		return nil, fmt.Errorf("%w: %d bytes, max: %d", ErrFileTooLarge, len(data), opts.MaxSize)
	}

	mimeType := opts.MimeType
	if mimeType == "" {
		mimeType = imageprocessor.DetectMimeType(data)
	}
	if !isValidMimeType(mimeType) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMimeType, mimeType)
	}

	// Хеш считается от ИСХОДНЫХ байтов, а не от результата оптимизации.
	// Благодаря этому ранний выход ниже срабатывает и при Optimize: true —
	// повторная заливка не перекодирует файл заново.
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	ext := filepath.Ext(opts.FileName)
	if ext == "" {
		ext = imageprocessor.GetExtensionByMimeType(mimeType)
	}
	key := buildKey(hash, ext)

	// Дедупликация: объект уже есть — не пишем.
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err == nil {
		width, height := s.processor.GetDimensionsFast(data)
		s.logger.Debug("object already exists, skipping upload",
			zap.String("key", key),
			zap.String("category", opts.Category.String()),
		)
		return &types.Photo{
			URL:      s.GetURL(key),
			FileName: opts.FileName,
			// Размер берётся из StatObject: на этом пути тело не вычислялось,
			// а в бакете может лежать оптимизированная версия, чей размер
			// отличается от len(data).
			Size:       info.Size,
			Width:      width,
			Height:     height,
			MimeType:   mimeType,
			Hash:       hash,
			StorageKey: key,
			UploadedAt: info.LastModified,
		}, nil
	}
	if !isNotFound(err) {
		return nil, fmt.Errorf("failed to stat object %q: %w", key, err)
	}

	body := data
	if opts.Optimize && isImage(mimeType) {
		optimized, optErr := s.processor.Optimize(data, mimeType)
		if optErr != nil {
			s.logger.Warn("failed to optimize image, uploading original",
				zap.String("key", key),
				zap.Error(optErr),
			)
		} else {
			body = optimized
		}
	}

	// Размеры берутся из исходных байтов: Optimize только перекодирует и
	// геометрию не меняет.
	width, height := s.processor.GetDimensionsFast(data)

	metadata := map[string]string{
		"category": opts.Category.String(),
	}
	if opts.FileName != "" {
		metadata["filename"] = opts.FileName
	}
	for k, v := range opts.Metadata {
		metadata[k] = v
	}

	if _, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(body), int64(len(body)),
		minio.PutObjectOptions{
			ContentType:  mimeType,
			UserMetadata: metadata,
		}); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUploadFailed, key, err)
	}

	return &types.Photo{
		URL:        s.GetURL(key),
		FileName:   opts.FileName,
		Size:       int64(len(body)),
		Width:      width,
		Height:     height,
		MimeType:   mimeType,
		Hash:       hash,
		StorageKey: key,
		UploadedAt: time.Now(),
	}, nil
}

// UploadMultiple загружает файлы параллельно, fail-fast.
func (s *MinIOStorage) UploadMultiple(ctx context.Context, files []FileUpload) (types.Photos, error) {
	photos := make(types.Photos, len(files))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxWorkers)

	for i, f := range files {
		g.Go(func() error {
			photo, err := s.Upload(ctx, f.Data, f.Options)
			if err != nil {
				return fmt.Errorf("file %d (%s): %w", i, f.Options.FileName, err)
			}
			// Запись по индексу: порядок сохраняется, пересортировка не нужна.
			photos[i] = *photo
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return photos, nil
}

// GetURL возвращает публичный URL объекта.
func (s *MinIOStorage) GetURL(storageKey string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, storageKey)
}

// Exists проверяет существование объекта.
func (s *MinIOStorage) Exists(ctx context.Context, storageKey string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, storageKey, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	if isNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to stat object %q: %w", storageKey, err)
}

// isNotFound отличает «объекта нет» от настоящей ошибки (сеть, доступ, 5xx).
// Спутать их нельзя: при ошибке доступа мы бы решили, что файла нет, и залили
// его заново.
func isNotFound(err error) bool {
	resp := minio.ToErrorResponse(err)
	return resp.StatusCode == http.StatusNotFound ||
		resp.Code == "NoSuchKey" ||
		resp.Code == "NoSuchBucket"
}

func isValidMimeType(mimeType string) bool {
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
		"video/mp4":  true,
		"video/webm": true,
	}
	return validTypes[mimeType]
}

func isImage(mimeType string) bool {
	return mimeType == "image/jpeg" ||
		mimeType == "image/png" ||
		mimeType == "image/gif" ||
		mimeType == "image/webp"
}
