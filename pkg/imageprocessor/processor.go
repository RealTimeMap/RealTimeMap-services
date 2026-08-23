package imageprocessor

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	"go.uber.org/zap"
)

type Processor struct {
	logger *zap.Logger
}

func NewProcessor(logger *zap.Logger) *Processor {
	return &Processor{logger: logger}
}

// GetDimensionsFast возвращает размеры БЕЗ полного декодирования изображения.
// Использует image.DecodeConfig, который читает только заголовок файла.
func (p *Processor) GetDimensionsFast(data []byte) (width, height int) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		p.logger.Warn("failed to decode image config", zap.Error(err))
		return 0, 0
	}
	return config.Width, config.Height
}

// Optimize оптимизирует изображение (сжатие).
// Важно: результат имеет ДРУГОЙ SHA256, чем вход. Ключ в хранилище считается
// от исходных байтов, см. pkg/storage.
func (p *Processor) Optimize(data []byte, mimeType string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	switch mimeType {
	case "image/jpeg", "image/jpg":
		// JPEG с качеством 85
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	case "image/png":
		// PNG без изменений (можно добавить pngquant)
		err = png.Encode(&buf, img)
	default:
		return data, nil
	}

	if err != nil {
		return nil, err
	}

	optimized := buf.Bytes()

	// Если оптимизированная версия больше - вернуть оригинал
	if len(optimized) > len(data) {
		return data, nil
	}

	p.logger.Info("image optimized",
		zap.Int("original_size", len(data)),
		zap.Int("optimized_size", len(optimized)),
		zap.Float64("saved_percent", float64(len(data)-len(optimized))/float64(len(data))*100),
	)

	return optimized, nil
}

// DetectMimeType определяет MIME тип по содержимому
func DetectMimeType(data []byte) string {
	if len(data) < 12 {
		return "application/octet-stream"
	}

	// JPEG
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// PNG
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// GIF
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 {
		return "image/gif"
	}

	// WebP
	if string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return "application/octet-stream"
}

// GetExtensionByMimeType возвращает расширение файла по MIME типу
func GetExtensionByMimeType(mimeType string) string {
	extensions := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/webp": ".webp",
		"video/mp4":  ".mp4",
		"video/webm": ".webm",
	}

	ext, ok := extensions[mimeType]
	if !ok {
		return ".bin"
	}
	return ext
}
