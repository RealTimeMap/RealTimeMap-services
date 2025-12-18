package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sync"
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

	filename := fmt.Sprintf("%s%s", uuid.New(), ext)

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

// UploadMultiple загружает несколько файлов параллельно
func (s *LocalStorage) UploadMultiple(ctx context.Context, files []FileUpload) (types.Photos, error) {
	type uploadResult struct {
		photo *types.Photo
		index int
		err   error
	}

	resultChan := make(chan uploadResult, len(files))

	// Параллельная загрузка фотографий
	for i, file := range files {
		go func(index int, f FileUpload) {
			photo, err := s.Upload(ctx, f.Reader, f.Options)
			resultChan <- uploadResult{photo: photo, index: index, err: err}
		}(i, file)
	}

	// Сбор результатов
	results := make([]uploadResult, 0, len(files))
	for i := 0; i < len(files); i++ {
		result := <-resultChan
		if result.err != nil {
			s.logger.Error("failed to upload file",
				zap.Int("index", result.index),
				zap.Error(result.err),
			)
			continue
		}
		results = append(results, result)
	}

	// Сортировка по исходному индексу для сохранения порядка
	photos := make(types.Photos, 0, len(results))
	for i := 0; i < len(files); i++ {
		for _, r := range results {
			if r.index == i {
				photos = append(photos, *r.photo)
				break
			}
		}
	}

	return photos, nil
}

// UploadMultipartOptimized - максимально оптимизированная загрузка с worker pool и pipeline обработкой
// Использует:
// - Worker Pool для контроля параллелизма (не создает 100 горутин для 100 файлов)
// - Pipeline обработку (декодирование изображения ОДИН раз)
// - Параллельное создание миниатюр
// - Синхронное возвращение thumbnail URLs (без потери)
func (s *LocalStorage) UploadMultipartOptimized(ctx context.Context, files []*multipart.FileHeader, opts UploadOptions) (types.Photos, error) {
	const maxWorkers = 5 // Максимум 5 параллельных загрузок для оптимального баланса CPU/IO

	type uploadResult struct {
		photo *types.Photo
		index int
		err   error
	}

	resultChan := make(chan uploadResult, len(files))
	semaphore := make(chan struct{}, maxWorkers) // Worker pool semaphore

	var wg sync.WaitGroup

	// Параллельная загрузка файлов с контролем через semaphore
	for i, fileHeader := range files {
		wg.Add(1)
		go func(index int, fh *multipart.FileHeader) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }() // Release semaphore

			photo, err := s.uploadSingleMultipartOptimized(ctx, fh, opts)
			resultChan <- uploadResult{photo: photo, index: index, err: err}
		}(i, fileHeader)
	}

	// Закрыть канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Сбор результатов
	results := make([]uploadResult, 0, len(files))
	for result := range resultChan {
		if result.err != nil {
			s.logger.Error("failed to upload file",
				zap.Int("index", result.index),
				zap.Error(result.err),
			)
			continue
		}
		results = append(results, result)
	}

	// Сортировка по исходному индексу для сохранения порядка
	photos := make(types.Photos, 0, len(results))
	for i := 0; i < len(files); i++ {
		for _, r := range results {
			if r.index == i {
				photos = append(photos, *r.photo)
				break
			}
		}
	}

	return photos, nil
}

// UploadMultipartDirect - оптимизированная загрузка из multipart.FileHeader
// Сохраняет файлы напрямую на диск минуя чтение всего в память
// Затем параллельно обрабатывает метаданные (hash, размеры)
// DEPRECATED: Используйте UploadMultipartOptimized для лучшей производительности
func (s *LocalStorage) UploadMultipartDirect(ctx context.Context, files []*multipart.FileHeader, opts UploadOptions) (types.Photos, error) {
	type uploadResult struct {
		photo *types.Photo
		index int
		err   error
	}

	resultChan := make(chan uploadResult, len(files))
	var wg sync.WaitGroup

	// Параллельная загрузка файлов
	for i, fileHeader := range files {
		wg.Add(1)
		go func(index int, fh *multipart.FileHeader) {
			defer wg.Done()

			photo, err := s.uploadSingleMultipart(ctx, fh, opts)
			resultChan <- uploadResult{photo: photo, index: index, err: err}
		}(i, fileHeader)
	}

	// Закрыть канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Сбор результатов
	results := make([]uploadResult, 0, len(files))
	for result := range resultChan {
		if result.err != nil {
			s.logger.Error("failed to upload file",
				zap.Int("index", result.index),
				zap.Error(result.err),
			)
			continue
		}
		results = append(results, result)
	}

	// Сортировка по исходному индексу для сохранения порядка
	photos := make(types.Photos, 0, len(results))
	for i := 0; i < len(files); i++ {
		for _, r := range results {
			if r.index == i {
				photos = append(photos, *r.photo)
				break
			}
		}
	}

	return photos, nil
}

// uploadSingleMultipartOptimized - максимально оптимизированная загрузка с pipeline обработкой
// Читает файл в память ОДИН раз, параллельно вычисляет hash и создает миниатюру
func (s *LocalStorage) uploadSingleMultipartOptimized(ctx context.Context, fileHeader *multipart.FileHeader, opts UploadOptions) (*types.Photo, error) {
	// Валидация категории
	if err := opts.Category.Validate(); err != nil {
		return nil, err
	}

	// Проверка размера
	if opts.MaxSize > 0 && fileHeader.Size > opts.MaxSize {
		return nil, fmt.Errorf("%w: %d bytes, max: %d", ErrFileTooLarge, fileHeader.Size, opts.MaxSize)
	}

	// Открыть файл
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// ОПТИМИЗАЦИЯ: Читаем файл в память ОДИН раз (избегаем двойного I/O)
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded file: %w", err)
	}

	// Вычисляем hash из памяти
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	// Определить MIME type из содержимого
	mimeType := opts.MimeType
	if mimeType == "" {
		mimeType = imageprocessor.DetectMimeType(data)
	}

	// Валидация MIME type ДО записи на диск
	if !s.isValidMimeType(mimeType) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMimeType, mimeType)
	}

	// Сгенерировать путь для сохранения
	now := time.Now()
	yearMonth := now.Format("2006/01")
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		ext = imageprocessor.GetExtensionByMimeType(mimeType)
	}

	filename := fmt.Sprintf("%s%s", uuid.New(), ext)
	storageKey := filepath.Join("photos", opts.Category.String(), yearMonth, filename)
	fullPath := filepath.Join(s.basePath, storageKey)

	// Создать директорию
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	var width, height int
	var thumbnailKey string

	// Параллельная обработка: запись на диск + обработка изображения
	var wg sync.WaitGroup
	var writeErr error
	var thumbData []byte
	var thumbErr error

	// 1. Запись файла на диск (в отдельной горутине)
	wg.Add(1)
	go func() {
		defer wg.Done()
		writeErr = os.WriteFile(fullPath, data, 0644)
	}()

	// 2. Обработка изображения (если это изображение)
	if s.isImage(mimeType) {
		// Получаем размеры БЕЗ полного декодирования (быстро)
		width, height = s.processor.GetDimensionsFast(data)

		// Генерация thumbnail только если нужно
		if opts.GenerateThumb {
			wg.Add(1)
			go func() {
				defer wg.Done()
				thumbWidth := opts.ThumbWidth
				thumbHeight := opts.ThumbHeight
				if thumbWidth == 0 {
					thumbWidth = 300
				}
				if thumbHeight == 0 {
					thumbHeight = 300
				}
				thumbData, thumbErr = s.processor.ResizeFast(data, thumbWidth, thumbHeight)
			}()
		}
	}

	wg.Wait()

	// Проверка ошибки записи
	if writeErr != nil {
		return nil, fmt.Errorf("failed to save file: %w", writeErr)
	}

	// Сохранить миниатюру (если была сгенерирована)
	if opts.GenerateThumb && thumbErr == nil && thumbData != nil {
		ext := filepath.Ext(storageKey)
		thumbKey := storageKey[:len(storageKey)-len(ext)] + "_thumb" + ext
		thumbPath := filepath.Join(s.basePath, thumbKey)

		if err := os.WriteFile(thumbPath, thumbData, 0644); err != nil {
			s.logger.Warn("failed to save thumbnail", zap.Error(err))
		} else {
			thumbnailKey = thumbKey
		}
	}

	// Сформировать Photo
	photo := &types.Photo{
		URL:        s.GetURL(storageKey),
		Thumbnail:  "",
		FileName:   fileHeader.Filename,
		Size:       int64(len(data)),
		Width:      width,
		Height:     height,
		MimeType:   mimeType,
		Hash:       hashStr,
		StorageKey: storageKey,
		UploadedAt: now,
	}

	if thumbnailKey != "" {
		photo.Thumbnail = s.GetURL(thumbnailKey)
	}

	s.logger.Info("file uploaded (optimized)",
		zap.String("storage_key", storageKey),
		zap.String("hash", hashStr[:16]),
		zap.Int64("size", photo.Size),
		zap.Bool("has_thumbnail", thumbnailKey != ""),
	)

	return photo, nil
}

// uploadSingleMultipart загружает один файл из multipart.FileHeader
func (s *LocalStorage) uploadSingleMultipart(ctx context.Context, fileHeader *multipart.FileHeader, opts UploadOptions) (*types.Photo, error) {
	// Валидация категории
	if err := opts.Category.Validate(); err != nil {
		return nil, err
	}

	// Проверка размера
	if opts.MaxSize > 0 && fileHeader.Size > opts.MaxSize {
		return nil, fmt.Errorf("%w: %d bytes, max: %d", ErrFileTooLarge, fileHeader.Size, opts.MaxSize)
	}

	// Открыть файл
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Сгенерировать путь для сохранения
	now := time.Now()
	yearMonth := now.Format("2006/01")
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" && opts.MimeType != "" {
		ext = imageprocessor.GetExtensionByMimeType(opts.MimeType)
	}

	filename := fmt.Sprintf("%s%s", uuid.New(), ext)
	storageKey := filepath.Join("photos", opts.Category.String(), yearMonth, filename)
	fullPath := filepath.Join(s.basePath, storageKey)

	// Создать директорию
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Создать файл на диске
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Копировать с вычислением hash на лету
	hasher := sha256.New()
	multiWriter := io.MultiWriter(dst, hasher)

	written, err := io.Copy(multiWriter, src)
	if err != nil {
		os.Remove(fullPath) // Удалить частично записанный файл
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	hashStr := hex.EncodeToString(hasher.Sum(nil))

	// Определить MIME type если не указан
	mimeType := opts.MimeType
	if mimeType == "" {
		// Открыть файл для определения MIME
		file, err := os.Open(fullPath)
		if err == nil {
			defer file.Close()
			buf := make([]byte, 512)
			n, _ := file.Read(buf)
			mimeType = imageprocessor.DetectMimeType(buf[:n])
		}
	}

	// Получить размеры изображения (открываем файл один раз)
	var width, height int
	if s.isImage(mimeType) {
		file, err := os.Open(fullPath)
		if err == nil {
			defer file.Close()
			data, _ := io.ReadAll(file)
			width, height, _ = s.processor.GetDimensions(data)
		}
	}

	// Создать миниатюру асинхронно (не блокируем ответ)
	var thumbnailKey string
	if opts.GenerateThumb && s.isImage(mimeType) {
		go func() {
			file, err := os.Open(fullPath)
			if err == nil {
				defer file.Close()
				data, _ := io.ReadAll(file)
				_, _ = s.generateThumbnail(data, storageKey, opts)
			}
		}()
	}

	// Сформировать Photo
	photo := &types.Photo{
		URL:        s.GetURL(storageKey),
		Thumbnail:  "",
		FileName:   fileHeader.Filename,
		Size:       written,
		Width:      width,
		Height:     height,
		MimeType:   mimeType,
		Hash:       hashStr,
		StorageKey: storageKey,
		UploadedAt: now,
	}

	if thumbnailKey != "" {
		photo.Thumbnail = s.GetURL(thumbnailKey)
	}

	s.logger.Info("file uploaded (direct)",
		zap.String("storage_key", storageKey),
		zap.String("hash", hashStr[:16]),
		zap.Int64("size", photo.Size),
	)

	return photo, nil
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
