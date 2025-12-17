package handlers

import (
	"fmt"
	"mime/multipart"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
)

const (
	maxFileSize      = 5 * 1024 * 1024 // 5 MB
	maxPhotosPerMark = 10
)

var allowedMimeTypes = []string{"image/jpeg", "image/png", "image/webp"}

// processPhotoUploads выполняет быструю валидацию файлов и возвращает headers для оптимизированной загрузки
// Это минимизирует задержку - не читаем файлы в память, просто валидируем headers
func processPhotoUploads(fileHeaders []*multipart.FileHeader) ([]*multipart.FileHeader, error) {
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

	// Быстрая валидация каждого файла (без чтения в память)
	if err := validatePhotoHeaders(fileHeaders); err != nil {
		return nil, err
	}

	return fileHeaders, nil
}

// validatePhotoHeaders проверяет заголовки загруженных файлов без чтения в память
// Это быстрая валидация перед передачей в storage
func validatePhotoHeaders(fileHeaders []*multipart.FileHeader) error {
	for i, fileHeader := range fileHeaders {
		// Проверка размера
		if fileHeader.Size > maxFileSize {
			return apperror.NewFieldValidationError(
				fmt.Sprintf("photos[%d]", i),
				fmt.Sprintf("file size exceeds maximum allowed size of %d MB", maxFileSize/(1024*1024)),
				"value_error.file.too_large",
				fileHeader.Size,
			)
		}

		// Проверка Content-Type header (предварительная)
		contentType := fileHeader.Header.Get("Content-Type")
		if !isAllowedContentType(contentType) {
			return apperror.NewInvalidMimeTypeError(
				fmt.Sprintf("photos[%d]", i),
				allowedMimeTypes,
				contentType,
			)
		}
	}

	return nil
}

// isAllowedContentType проверяет допустимость типа файла
func isAllowedContentType(contentType string) bool {
	for _, allowed := range allowedMimeTypes {
		if allowed == contentType {
			return true
		}
	}
	return false
}
