package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

func Load[T any](opts ...Option) (*T, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	var cfg T

	// Ищем первый существующий конфиг файл
	configPath := findConfigFile(options.Paths)

	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
	} else if !options.UseEnv {
		return nil, ErrConfigNotFound
	}

	// Загружаем/перезаписываем из ENV
	if options.UseEnv {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("failed to read env: %w", err)
		}
	}

	return &cfg, nil
}

// MustLoad загружает конфиг или паникует
func MustLoad[T any](opts ...Option) *T {
	cfg, err := Load[T](opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func findConfigFile(paths []string) string {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// GetEnv возвращает значение ENV или дефолт
func GetEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// MustGetEnv возвращает значение ENV или паникует
func MustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required env variable %s is not set", key))
	}
	return val
}
