package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Photo struct {
	URL        string    `json:"url"`                   // URL или путь к файлу на сервере
	Thumbnail  string    `json:"thumbnail,omitempty"`   // URL миниатюры (опционально)
	FileName   string    `json:"file_name,omitempty"`   // Оригинальное имя файла
	Size       int64     `json:"size,omitempty"`        // Размер файла в байтах
	Width      int       `json:"width,omitempty"`       // Ширина изображения в пикселях
	Height     int       `json:"height,omitempty"`      // Высота изображения в пикселях
	MimeType   string    `json:"mime_type,omitempty"`   // MIME тип (image/jpeg, image/png, image/webp и т.д.)
	Hash       string    `json:"hash,omitempty"`        // Хэш файла (SHA256) для проверки целостности
	StorageKey string    `json:"storage_key,omitempty"` // Ключ/путь в хранилище (S3, MinIO и т.д.)
	UploadedAt time.Time `json:"uploaded_at,omitempty"` // Время загрузки
}

// Scan реализует интерфейс sql.Scanner для чтения Photo из БД
func (p *Photo) Scan(val interface{}) error {
	if val == nil {
		*p = Photo{}
		return nil
	}

	var data []byte
	switch v := val.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into Photo", val)
	}

	return json.Unmarshal(data, p)
}

// Value реализует интерфейс driver.Valuer для записи Photo в БД
func (p Photo) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Photos представляет массив фотографий
type Photos []Photo

// Scan реализует интерфейс sql.Scanner для чтения Photos из БД
func (p *Photos) Scan(val interface{}) error {
	if val == nil {
		*p = Photos{}
		return nil
	}

	var data []byte
	switch v := val.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into Photos", val)
	}

	return json.Unmarshal(data, p)
}

// Value реализует интерфейс driver.Valuer для записи Photos в БД
func (p Photos) Value() (driver.Value, error) {
	if len(p) == 0 {
		return nil, nil
	}

	return json.Marshal(p)
}
