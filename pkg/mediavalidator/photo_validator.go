package mediavalidator

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	_ "golang.org/x/image/webp"
)

const (
	DefaultMaxFileSize      = 5 << 20 // 5 мб
	DefaultMaxFilePerUpload = 10
)

var DefaultAllowedMimeTypes = []string{
	"image/jpeg",
	"image/png",
	"image/webp",
}

type PhotoInput struct {
	Data     []byte
	FileName string
}

type PhotoValidator struct {
	maxFileSize      int64
	maxFileCount     int
	allowedMimeTypes []string
}

type Config struct {
	MaxFileSize      int64
	MaxFileCount     int
	AllowedMimeTypes []string
}

func NewPhotoValidator() *PhotoValidator {
	return &PhotoValidator{
		maxFileSize:      DefaultMaxFileSize,
		maxFileCount:     DefaultMaxFilePerUpload,
		allowedMimeTypes: DefaultAllowedMimeTypes,
	}
}

func NewPhotoValidatorWithConfig(config *Config) *PhotoValidator {
	v := NewPhotoValidator()

	if config.MaxFileSize > 0 {
		v.maxFileSize = config.MaxFileSize
	}
	if config.MaxFileCount > 0 {
		v.maxFileCount = config.MaxFileCount
	}
	if len(config.AllowedMimeTypes) > 0 {
		v.allowedMimeTypes = config.AllowedMimeTypes
	}
	return v
}

// PhotoValidationError обёртка над validation.ValidationError для реализации интерфейса error
type PhotoValidationError struct {
	ValidationError validation.ValidationError
}

func (e PhotoValidationError) Error() string {
	return e.ValidationError.Msg
}

// ToValidationError возвращает validation.ValidationError для использования в handler
func (e PhotoValidationError) ToValidationError() validation.ValidationError {
	return e.ValidationError
}

// ValidatePhotos валидирует список фотографий
// Возвращает PhotoValidationError при ошибке
func (v *PhotoValidator) ValidatePhotos(photos []PhotoInput) error {
	// Проверка количества
	if len(photos) > v.maxFileCount {
		return PhotoValidationError{
			ValidationError: validation.NewFieldError(
				"photos",
				fmt.Sprintf("too many photos. Maximum allowed: %d, received: %d", v.maxFileCount, len(photos)),
				"value_error.list.max_items",
				len(photos),
			),
		}
	}

	// Валидация каждого фото
	for i, photo := range photos {
		if err := v.validateSinglePhoto(i, photo); err != nil {
			return err
		}
	}

	return nil
}

// validateSinglePhoto валидирует одно фото
func (v *PhotoValidator) validateSinglePhoto(index int, photo PhotoInput) error {
	fieldName := fmt.Sprintf("photos[%d]", index)

	// 1. Проверка размера
	fileSize := int64(len(photo.Data))
	if fileSize > v.maxFileSize {
		return PhotoValidationError{
			ValidationError: validation.NewFieldError(
				fieldName,
				fmt.Sprintf("file size exceeds maximum allowed size of %d bytes (%d MB)",
					v.maxFileSize,
					v.maxFileSize/(1024*1024)),
				"value_error.file.too_large",
				fileSize,
			),
		}
	}

	// 2. Проверка реального MIME type из байтов (НЕ из HTTP заголовка!)
	mimeType := http.DetectContentType(photo.Data)
	if !v.isAllowedMimeType(mimeType) {
		return PhotoValidationError{
			ValidationError: validation.NewFieldError(
				fieldName,
				fmt.Sprintf("file type not allowed. Allowed types: jpeg, png, webp. Received: %s", mimeType),
				"value_error.mime_type",
				mimeType,
			),
		}
	}

	// 3. Проверка что изображение можно декодировать
	if _, _, err := image.Decode(bytes.NewReader(photo.Data)); err != nil {
		return PhotoValidationError{
			ValidationError: validation.NewFieldError(
				fieldName,
				"invalid image: cannot decode",
				"value_error.image",
				nil,
			),
		}
	}

	return nil
}

// isAllowedMimeType проверяет разрешён ли MIME тип
func (v *PhotoValidator) isAllowedMimeType(mimeType string) bool {
	for _, allowed := range v.allowedMimeTypes {
		if mimeType == allowed {
			return true
		}
	}
	return false
}

// ValidateSinglePhoto валидирует одно фото (публичный метод)
func (v *PhotoValidator) ValidateSinglePhoto(photo PhotoInput) error {
	return v.validateSinglePhoto(0, photo)
}

// GetMaxFileSize возвращает максимальный размер файла
func (v *PhotoValidator) GetMaxFileSize() int64 {
	return v.maxFileSize
}

// GetMaxPhotosCount возвращает максимальное количество фото
func (v *PhotoValidator) GetMaxPhotosCount() int {
	return v.maxFileCount
}

// GetAllowedMimeTypes возвращает список разрешённых MIME типов
func (v *PhotoValidator) GetAllowedMimeTypes() []string {
	return v.allowedMimeTypes
}
