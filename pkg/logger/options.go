package logger

import (
	"os"

	"go.uber.org/zap/zapcore"
)

type Options struct {
	Level       zapcore.Level
	Encoding    string // "json" или "console"
	Output      zapcore.WriteSyncer
	AddCaller   bool
	AddStack    zapcore.LevelEnabler
	ServiceName string
}

type Option func(*Options)

func defaultOptions() *Options {
	return &Options{
		Level:     zapcore.InfoLevel,
		Encoding:  "json",
		Output:    zapcore.AddSync(os.Stdout),
		AddCaller: true,
		AddStack:  zapcore.ErrorLevel,
	}
}

// WithLevel устанавливает уровень логирования
func WithLevel(level string) Option {
	return func(o *Options) {
		switch level {
		case "debug":
			o.Level = zapcore.DebugLevel
		case "info":
			o.Level = zapcore.InfoLevel
		case "warn":
			o.Level = zapcore.WarnLevel
		case "error":
			o.Level = zapcore.ErrorLevel
		default:
			o.Level = zapcore.InfoLevel
		}
	}
}

// WithEncoding устанавливает формат вывода
func WithEncoding(encoding string) Option {
	return func(o *Options) {
		if encoding == "console" || encoding == "json" {
			o.Encoding = encoding
		}
	}
}

// WithOutput устанавливает куда писать логи
func WithOutput(w zapcore.WriteSyncer) Option {
	return func(o *Options) {
		o.Output = w
	}
}

// WithServiceName добавляет имя сервиса ко всем логам
func WithServiceName(name string) Option {
	return func(o *Options) {
		o.ServiceName = name
	}
}

// WithCaller включает/выключает информацию о месте вызова
func WithCaller(enabled bool) Option {
	return func(o *Options) {
		o.AddCaller = enabled
	}
}

// WithStacktrace устанавливает уровень для стектрейсов
func WithStacktrace(level zapcore.LevelEnabler) Option {
	return func(o *Options) {
		o.AddStack = level
	}
}
