package imageprocessor

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"sync"

	"github.com/disintegration/imaging"
	"go.uber.org/zap"
)

// Буфер pool для уменьшения аллокаций памяти
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type Processor struct {
	logger *zap.Logger
}

func NewProcessor(logger *zap.Logger) *Processor {
	return &Processor{logger: logger}
}

// DecodeImage декодирует изображение из байтов (вызывается один раз)
func (p *Processor) DecodeImage(data []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}
	return img, format, nil
}

// GetDimensions метод возвращает ширину и высоту картинки (полное декодирование)
func (p *Processor) GetDimensions(data []byte) (width, height int, err error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}

// GetDimensionsFast возвращает размеры БЕЗ полного декодирования изображения
// Использует image.DecodeConfig который читает только заголовок файла
func (p *Processor) GetDimensionsFast(data []byte) (width, height int) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		p.logger.Warn("failed to decode image config", zap.Error(err))
		return 0, 0
	}
	return config.Width, config.Height
}

// GetDimensionsFromDecoded получает размеры из уже декодированного изображения (без повторного декодирования)
func (p *Processor) GetDimensionsFromDecoded(img image.Image) (width, height int) {
	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy()
}

// Resize изменяет размер изображения (качественный, но медленный - Lanczos)
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

// ResizeFast быстрое изменение размера для thumbnails (Box алгоритм вместо Lanczos)
// Box в ~3-4 раза быстрее Lanczos и достаточен для превью
func (p *Processor) ResizeFast(data []byte, width, height int) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Box filter - быстрый алгоритм, достаточный для thumbnails
	resized := imaging.Fit(img, width, height, imaging.Box)

	// Используем buffer pool
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	// Encode с уменьшенным качеством для thumbnails (быстрее + меньше размер)
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 75})
	case "png":
		// PNG конвертируем в JPEG для thumbnails (значительно быстрее)
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 75})
	case "gif":
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 75})
	default:
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 75})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	// Копируем данные из buffer pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// ResizeFromDecoded изменяет размер уже декодированного изображения (без повторного декодирования)
// format - оригинальный формат изображения ("jpeg", "png", etc.)
func (p *Processor) ResizeFromDecoded(img image.Image, format string, width, height int) ([]byte, error) {
	// Resize с сохранением пропорций
	resized := imaging.Fit(img, width, height, imaging.Lanczos)

	// Используем buffer pool
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	// Encode
	var err error
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(buf, resized)
	case "gif":
		err = gif.Encode(buf, resized, nil)
	default:
		err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	// Копируем данные из buffer pool перед возвратом
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
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

// OptimizeFromDecoded оптимизирует уже декодированное изображение (без повторного декодирования)
func (p *Processor) OptimizeFromDecoded(img image.Image, mimeType string, originalSize int) ([]byte, error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	var err error
	switch mimeType {
	case "image/jpeg", "image/jpg":
		err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})
	case "image/png":
		err = png.Encode(buf, img)
	default:
		return nil, fmt.Errorf("unsupported mime type for optimization: %s", mimeType)
	}

	if err != nil {
		return nil, err
	}

	optimized := make([]byte, buf.Len())
	copy(optimized, buf.Bytes())

	// Логируем только если действительно сжали
	if originalSize > 0 && len(optimized) < originalSize {
		p.logger.Info("image optimized",
			zap.Int("original_size", originalSize),
			zap.Int("optimized_size", len(optimized)),
			zap.Float64("saved_percent", float64(originalSize-len(optimized))/float64(originalSize)*100),
		)
	}

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
