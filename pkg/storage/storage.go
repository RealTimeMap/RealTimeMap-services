package storage

import (
	"context"
	"io"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

type Storage interface {
	// Upload загружает файл и возвращает Photo с метаданными
	Upload(ctx context.Context, file io.Reader, opts UploadOptions) (*types.Photo, error)

	// UploadMultiple загружает несколько файлов
	UploadMultiple(ctx context.Context, files []FileUpload) (types.Photos, error)

	// Delete удаляет файл по storage key
	Delete(ctx context.Context, storageKey string) error

	// DeleteMultiple удаляет несколько файлов
	DeleteMultiple(ctx context.Context, storageKeys []string) error

	// GetURL возвращает публичный URL файла
	GetURL(storageKey string) string

	// GetSignedURL возвращает подписанный URL (для приватных файлов)
	GetSignedURL(ctx context.Context, storageKey string, expiration time.Duration) (string, error)

	// Exists проверяет существование файла
	Exists(ctx context.Context, storageKey string) (bool, error)
}
type CategoryStorage string

const (
	CategoryMarkPhoto    CategoryStorage = "marks"
	CategoryCommentPhoto CategoryStorage = "comments"
	CategoryTemp         CategoryStorage = "temp"
	CategoryCategories   CategoryStorage = "categories"
)

// String строковое представление
func (c CategoryStorage) String() string {
	return string(c)
}

// Validate проверяет валидность категории, если вашей нет, добавить выше
func (c CategoryStorage) Validate() error {
	switch c {
	case CategoryMarkPhoto, CategoryCommentPhoto, CategoryTemp, CategoryCategories:
		return nil
	default:
		return ErrInvalidCategory
	}
}

type UploadOptions struct {
	FileName      string            // Исходное название файла
	Category      CategoryStorage   // Категория хранения
	MimeType      string            // MIME тип
	MaxSize       int64             // Максимальный размер (байты)
	GenerateThumb bool              // Генерировать миниатюру
	ThumbWidth    int               // Ширина миниатюры
	ThumbHeight   int               // Высота миниатюры
	Optimize      bool              // Оптимизировать изображение
	Metadata      map[string]string // Дополнительные метаданные
}

type FileUpload struct {
	Reader  io.Reader
	Options UploadOptions
}

// StorageConfig - конфигурация хранилища
type StorageConfig struct {
	Type     string `yaml:"type" env:"STORAGE_TYPE"`           // local, s3, minio
	BasePath string `yaml:"base_path" env:"STORAGE_BASE_PATH"` // Для local
	BaseURL  string `yaml:"base_url" env:"STORAGE_BASE_URL"`   // Публичный URL

	Endpoint        string `yaml:"endpoint" env:"STORAGE_ENDPOINT"`
	Bucket          string `yaml:"bucket" env:"STORAGE_BUCKET"`
	AccessKeyID     string `yaml:"access_key_id" env:"STORAGE_ACCESS_KEY_ID"`
	SecretAccessKey string `yaml:"secret_access_key" env:"STORAGE_SECRET_ACCESS_KEY"`
	Region          string `yaml:"region" env:"STORAGE_REGION"`
	UseSSL          bool   `yaml:"use_ssl" env:"STORAGE_USE_SSL"`
}
