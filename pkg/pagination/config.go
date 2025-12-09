package pagination

// Config определяет ограничения
type Config struct {
	DefaultPageSize int
	MinPageSize     int
	MaxPageSize     int
}

// DefaultConfig - конфигурация по умолчанию
var DefaultConfig = Config{
	DefaultPageSize: 10,
	MinPageSize:     1,
	MaxPageSize:     100,
}

// NewConfig создает кастомную конфигурацию
func NewConfig(defaultSize, minSize, maxSize int) Config {
	return Config{
		DefaultPageSize: defaultSize,
		MinPageSize:     minSize,
		MaxPageSize:     maxSize,
	}
}
