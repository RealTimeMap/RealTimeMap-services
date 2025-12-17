package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/imageprocessor"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LocalStorage - реализация хранилища на локальной файловой системе
type LocalStorage struct {
	basePath  string // /var/www/storage или ./storage
	baseURL   string // http://localhost:8080/uploads
	processor *imageprocessor.Processor
	logger    *zap.Logger
}

// NewLocalStorage создает LocalStorage
func NewLocalStorage(basePath, baseURL string, logger *zap.Logger) (Storage, error) {
	// Создать базовую директорию если не существует
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	// Создать поддиректории для категорий
	categories := []CategoryStorage{
		CategoryMarkPhoto,
		CategoryCommentPhoto,
		CategoryTemp,
	}

	for _, cat := range categories {
		catPath := filepath.Join(basePath, "photos", cat.String())
		if err := os.MkdirAll(catPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create category path %s: %w", cat, err)
		}
	}

	processor := imageprocessor.NewProcessor(logger)

	return &LocalStorage{
		basePath:  basePath,
		baseURL:   baseURL,
		processor: processor,
		logger:    logger,
	}, nil
}

// Upload загружает файл
func (s *LocalStorage) Upload(ctx context.Context, file io.Reader, opts UploadOptions) (*types.Photo, error) {
	// Валидация категории
	if err := opts.Category.Validate(); err != nil {
		return nil, err
	}

	// Прочитать файл в память (для вычисления хеша и обработки)
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Проверка размера
	if opts.MaxSize > 0 && int64(len(data)) > opts.MaxSize {
		return nil, fmt.Errorf("%w: %d bytes, max: %d", ErrFileTooLarge, len(data), opts.MaxSize)
	}

	// Вычислить SHA256 хеш
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	// Определить MIME type если не указан
	if opts.MimeType == "" {
		opts.MimeType = imageprocessor.DetectMimeType(data)
	}

	// Валидация MIME type
	if !s.isValidMimeType(opts.MimeType) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMimeType, opts.MimeType)
	}

	// Оптимизация изображения
	if opts.Optimize && s.isImage(opts.MimeType) {
		optimized, err := s.processor.Optimize(data, opts.MimeType)
		if err != nil {
			s.logger.Warn("failed to optimize image", zap.Error(err))
		} else {
			data = optimized
		}
	}

	// Получить размеры изображения
	width, height, err := s.processor.GetDimensions(data)
	if err != nil {
		s.logger.Warn("failed to get image dimensions", zap.Error(err))
	}

	// Сгенерировать путь: photos/marks/2024/01/abc123def.jpg
	now := time.Now()
	yearMonth := now.Format("2006/01")
	ext := filepath.Ext(opts.FileName)
	if ext == "" {
		ext = imageprocessor.GetExtensionByMimeType(opts.MimeType)
	}

	filename := fmt.Sprintf("%s%s", uuid.New(), ext) // TODO пофиксить мб на uuid

	storageKey := filepath.Join("photos", opts.Category.String(), yearMonth, filename)
	fullPath := filepath.Join(s.basePath, storageKey)

	// Создать директорию
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Проверка: файл уже существует? (дедупликация по хешу)
	if _, err := os.Stat(fullPath); err == nil {
		s.logger.Info("file already exists, reusing", zap.String("path", storageKey))
	} else {
		// Сохранить файл
		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
	}

	// Создать миниатюру
	var thumbnailKey string
	if opts.GenerateThumb && s.isImage(opts.MimeType) {
		thumbnailKey, err = s.generateThumbnail(data, storageKey, opts)
		if err != nil {
			s.logger.Warn("failed to generate thumbnail", zap.Error(err))
		}
	}

	// Сформировать Photo
	photo := &types.Photo{
		URL:        s.GetURL(storageKey),
		Thumbnail:  "",
		FileName:   opts.FileName,
		Size:       int64(len(data)),
		Width:      width,
		Height:     height,
		MimeType:   opts.MimeType,
		Hash:       hashStr,
		StorageKey: storageKey,
		UploadedAt: now,
	}

	if thumbnailKey != "" {
		photo.Thumbnail = s.GetURL(thumbnailKey)
	}

	s.logger.Info("file uploaded",
		zap.String("storage_key", storageKey),
		zap.String("hash", hashStr[:16]),
		zap.Int64("size", photo.Size),
	)

	return photo, nil
}

// UploadMultiple загружает несколько файлов
func (s *LocalStorage) UploadMultiple(ctx context.Context, files []FileUpload) (types.Photos, error) {
	photos := make(types.Photos, 0, len(files))

	for i, file := range files {
		photo, err := s.Upload(ctx, file.Reader, file.Options)
		if err != nil {
			s.logger.Error("failed to upload file",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}
		photos = append(photos, *photo)
	}

	return photos, nil
}

// Delete удаляет файл
func (s *LocalStorage) Delete(ctx context.Context, storageKey string) error {
	fullPath := filepath.Join(s.basePath, storageKey)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Удалить миниатюру если есть
	thumbPath := s.getThumbnailPath(storageKey)
	if _, err := os.Stat(thumbPath); err == nil {
		os.Remove(thumbPath)
	}

	s.logger.Info("file deleted", zap.String("storage_key", storageKey))
	return nil
}

// DeleteMultiple удаляет несколько файлов
func (s *LocalStorage) DeleteMultiple(ctx context.Context, storageKeys []string) error {
	var errs []error
	for _, key := range storageKeys {
		if err := s.Delete(ctx, key); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to delete %d files", len(errs))
	}

	return nil
}

// GetURL возвращает публичный URL файла
func (s *LocalStorage) GetURL(storageKey string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, storageKey)
}

// GetSignedURL для local storage возвращает обычный URL
func (s *LocalStorage) GetSignedURL(ctx context.Context, storageKey string, expiration time.Duration) (string, error) {
	return s.GetURL(storageKey), nil
}

// Exists проверяет существование файла
func (s *LocalStorage) Exists(ctx context.Context, storageKey string) (bool, error) {
	fullPath := filepath.Join(s.basePath, storageKey)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// generateThumbnail создает миниатюру
func (s *LocalStorage) generateThumbnail(data []byte, originalKey string, opts UploadOptions) (string, error) {
	width := opts.ThumbWidth
	height := opts.ThumbHeight
	if width == 0 {
		width = 300
	}
	if height == 0 {
		height = 300
	}

	thumbData, err := s.processor.Resize(data, width, height)
	if err != nil {
		return "", err
	}

	// Путь миниатюры: добавить _thumb перед расширением
	ext := filepath.Ext(originalKey)
	thumbKey := originalKey[:len(originalKey)-len(ext)] + "_thumb" + ext
	thumbPath := filepath.Join(s.basePath, thumbKey)

	if err := os.WriteFile(thumbPath, thumbData, 0644); err != nil {
		return "", err
	}

	return thumbKey, nil
}

// getThumbnailPath возвращает путь к миниатюре
func (s *LocalStorage) getThumbnailPath(originalKey string) string {
	ext := filepath.Ext(originalKey)
	thumbKey := originalKey[:len(originalKey)-len(ext)] + "_thumb" + ext
	return filepath.Join(s.basePath, thumbKey)
}

// isValidMimeType проверяет валидность MIME типа
func (s *LocalStorage) isValidMimeType(mimeType string) bool {
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

// isImage проверяет, является ли MIME тип изображением
func (s *LocalStorage) isImage(mimeType string) bool {
	return mimeType == "image/jpeg" ||
		mimeType == "image/png" ||
		mimeType == "image/gif" ||
		mimeType == "image/webp"
}
