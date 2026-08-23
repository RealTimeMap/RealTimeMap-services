package storage

import (
	"context"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
)

// Storage — content-addressed хранилище файлов.
//
// Ключ объекта выводится из SHA256 ИСХОДНЫХ байтов, поэтому один и тот же файл,
// загруженный дважды (в том числе из разных сервисов), занимает место один раз
// и получает один и тот же ключ. Прямое следствие: объект разделяется между
// сущностями, и удаления в интерфейсе нет — см. решения 7 и 8 в
// .scratch/storage-rewrite/spec.md.
type Storage interface {
	// Upload загружает файл. data — полные байты; при лимите в 5 МБ стриминг
	// не нужен, а content-addressed ключ требует хеша до записи.
	Upload(ctx context.Context, data []byte, opts UploadOptions) (*types.Photo, error)

	// UploadMultiple загружает несколько файлов. Fail-fast: при первой ошибке
	// остальные отменяются и возвращается error, частичный результат не
	// отдаётся.
	UploadMultiple(ctx context.Context, files []FileUpload) (types.Photos, error)

	// GetURL возвращает публичный URL объекта.
	GetURL(storageKey string) string

	// Exists проверяет существование объекта.
	Exists(ctx context.Context, storageKey string) (bool, error)
}

type CategoryStorage string

const (
	CategoryMarkPhoto    CategoryStorage = "marks"
	CategoryCommentPhoto CategoryStorage = "comments"
	// CategoryTemp нигде не используется. Учти: категория больше не входит в
	// ключ объекта (решение 12), поэтому повесить на неё lifecycle-политику по
	// префиксу нельзя.
	CategoryTemp          CategoryStorage = "temp"
	CategoryCategories    CategoryStorage = "categories"
	CategoryProfileAvatar CategoryStorage = "avatars"
	CategoryAchievement   CategoryStorage = "achievements"
)

// String строковое представление
func (c CategoryStorage) String() string {
	return string(c)
}

// Validate проверяет валидность категории, если вашей нет, добавить выше
func (c CategoryStorage) Validate() error {
	switch c {
	case CategoryMarkPhoto, CategoryCommentPhoto, CategoryTemp, CategoryCategories, CategoryProfileAvatar, CategoryAchievement:
		return nil
	default:
		return ErrInvalidCategory
	}
}

type UploadOptions struct {
	FileName string          // Исходное имя файла, идёт в user-metadata
	Category CategoryStorage // Категория хранения, идёт в user-metadata
	MimeType string          // Если пусто — определяется по содержимому
	MaxSize  int64           // Максимальный размер (байты)
	// Optimize перекодирует изображение перед записью. Применяется только при
	// первой заливке: если объект с таким ключом уже есть, запись не
	// происходит вовсе.
	Optimize bool
	Metadata map[string]string // Дополнительные user-metadata
}

type FileUpload struct {
	Data    []byte
	Options UploadOptions
}

// StorageConfig - конфигурация хранилища
type StorageConfig struct {
	Endpoint        string `yaml:"endpoint" env:"STORAGE_ENDPOINT"`
	Bucket          string `yaml:"bucket" env:"STORAGE_BUCKET"`
	AccessKeyID     string `yaml:"access_key_id" env:"STORAGE_ACCESS_KEY_ID"`
	SecretAccessKey string `yaml:"secret_access_key" env:"STORAGE_SECRET_ACCESS_KEY"`
	Region          string `yaml:"region" env:"STORAGE_REGION"`
	UseSSL          bool   `yaml:"use_ssl" env:"STORAGE_USE_SSL"`
	// BaseURL — публичный префикс раздачи. Обязан быть одинаковым во всех
	// сервисах, иначе один объект получит разные URL.
	BaseURL string `yaml:"base_url" env:"STORAGE_BASE_URL"`
}
