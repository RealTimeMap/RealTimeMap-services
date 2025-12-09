package imageprocessor

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	"github.com/disintegration/imaging"
	"go.uber.org/zap"
)

type Processor struct {
	logger *zap.Logger
}

func NewProcessor(logger *zap.Logger) *Processor {
	return &Processor{logger: logger}
}

// GetDimensions метод возвращает ширину и высоту картинки
func (p *Processor) GetDimensions(data []byte) (width, height int, err error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}

// Resize изменяет размер изображения
func (p *Processor) Resize(data []byte, width, height int) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize с сохранением пропорций
	resized := imaging.Fit(img, width, height, imaging.Lanczos)

	// Encode обратно
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(&buf, resized)
	case "gif":
		err = gif.Encode(&buf, resized, nil)
	default:
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return buf.Bytes(), nil
}

// Optimize оптимизирует изображение (сжатие)
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

// Validate валидирует изображение
func (p *Processor) Validate(data []byte, maxWidth, maxHeight int) error {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if maxWidth > 0 && width > maxWidth {
		return fmt.Errorf("image width %d exceeds maximum %d", width, maxWidth)
	}

	if maxHeight > 0 && height > maxHeight {
		return fmt.Errorf("image height %d exceeds maximum %d", height, maxHeight)
	}

	return nil
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
