package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
)

const (
	maxFileSize      = 5 * 1024 * 1024 // 5 MB
	maxPhotosPerMark = 10
)

var allowedMimeTypes = []string{"image/jpeg", "image/png", "image/webp"}

// processPhotoUploads читает файлы в память параллельно и валидирует их
// Возвращает чистые данные []PhotoInput для передачи в Service Layer (Clean Architecture)
func processPhotoUploads(fileHeaders []*multipart.FileHeader) ([]mediavalidator.PhotoInput, error) {
	if len(fileHeaders) == 0 {
		return nil, nil
	}

	// Проверка количества
	if len(fileHeaders) > maxPhotosPerMark {
		return nil, apperror.NewFieldValidationError(
			"photos",
			fmt.Sprintf("too many photos. Maximum allowed: %d, received: %d", maxPhotosPerMark, len(fileHeaders)),
			"value_error.list.max_items",
			len(fileHeaders),
		)
	}

	// Параллельное чтение файлов в память
	photos, err := readFilesParallel(fileHeaders)
	if err != nil {
		return nil, err
	}

	return photos, nil
}

// readFilesParallel читает файлы параллельно и валидирует MIME type из реальных байтов
func readFilesParallel(fileHeaders []*multipart.FileHeader) ([]mediavalidator.PhotoInput, error) {
	type readResult struct {
		photo mediavalidator.PhotoInput
		index int
		err   error
	}

	resultChan := make(chan readResult, len(fileHeaders))
	var wg sync.WaitGroup

	// Параллельное чтение
	for i, fh := range fileHeaders {
		wg.Add(1)
		go func(index int, header *multipart.FileHeader) {
			defer wg.Done()

			photo, err := readSingleFile(index, header)
			resultChan <- readResult{photo: photo, index: index, err: err}
		}(i, fh)
	}

	// Закрыть канал после завершения
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Сбор результатов с сохранением порядка
	results := make([]readResult, 0, len(fileHeaders))
	for result := range resultChan {
		if result.err != nil {
			return nil, result.err // Возвращаем первую ошибку
		}
		results = append(results, result)
	}

	// Сортировка по индексу для сохранения порядка
	photos := make([]mediavalidator.PhotoInput, len(fileHeaders))
	for _, r := range results {
		photos[r.index] = r.photo
	}

	return photos, nil
}

// readSingleFile читает один файл и валидирует его
func readSingleFile(index int, header *multipart.FileHeader) (mediavalidator.PhotoInput, error) {
	// Проверка размера
	if header.Size > maxFileSize {
		return mediavalidator.PhotoInput{}, apperror.NewFieldValidationError(
			fmt.Sprintf("photos[%d]", index),
			fmt.Sprintf("file size exceeds maximum allowed size of %d MB", maxFileSize/(1024*1024)),
			"value_error.file.too_large",
			header.Size,
		)
	}

	// Открыть файл
	file, err := header.Open()
	if err != nil {
		return mediavalidator.PhotoInput{}, apperror.NewFieldValidationError(
			fmt.Sprintf("photos[%d]", index),
			"failed to open uploaded file",
			"value_error.file.open",
			nil,
		)
	}
	defer file.Close()

	// Читаем в память
	data, err := io.ReadAll(file)
	if err != nil {
		return mediavalidator.PhotoInput{}, apperror.NewFieldValidationError(
			fmt.Sprintf("photos[%d]", index),
			"failed to read uploaded file",
			"value_error.file.read",
			nil,
		)
	}

	// Валидация MIME type из реальных байтов (не из HTTP заголовка!)
	mimeType := http.DetectContentType(data)
	if !isAllowedMimeType(mimeType) {
		return mediavalidator.PhotoInput{}, apperror.NewInvalidMimeTypeError(
			fmt.Sprintf("photos[%d]", index),
			allowedMimeTypes,
			mimeType,
		)
	}

	return mediavalidator.PhotoInput{
		Data:     data,
		FileName: header.Filename,
	}, nil
}

// isAllowedMimeType проверяет допустимость MIME типа
func isAllowedMimeType(mimeType string) bool {
	for _, allowed := range allowedMimeTypes {
		if allowed == mimeType {
			return true
		}
	}
	return false
}
