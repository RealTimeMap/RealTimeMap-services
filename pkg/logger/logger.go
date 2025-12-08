package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
)

// New создаёт новый логгер с переданными опциями
func New(opts ...Option) *zap.Logger {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if options.Encoding == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, options.Output, options.Level)

	zapOpts := []zap.Option{}
	if options.AddCaller {
		zapOpts = append(zapOpts, zap.AddCaller())
	}
	if options.AddStack != nil {
		zapOpts = append(zapOpts, zap.AddStacktrace(options.AddStack))
	}

	logger := zap.New(core, zapOpts...)

	if options.ServiceName != "" {
		logger = logger.With(zap.String("service", options.ServiceName))
	}

	return logger
}

// NewByEnv создаёт логгер на основе окружения
func NewByEnv(env string, serviceName string) *zap.Logger {
	switch env {
	case EnvLocal:
		return New(
			WithLevel("debug"),
			WithEncoding("console"),
			WithServiceName(serviceName),
		)
	case EnvDev:
		return New(
			WithLevel("debug"),
			WithEncoding("json"),
			WithServiceName(serviceName),
		)
	case EnvProd:
		return New(
			WithLevel("info"),
			WithEncoding("json"),
			WithServiceName(serviceName),
			WithStacktrace(zapcore.ErrorLevel),
		)
	default:
		return New(
			WithLevel("info"),
			WithServiceName(serviceName),
		)
	}
}

// MustNewByEnv создаёт логгер и устанавливает глобальный
func MustNewByEnv(env string, serviceName string) *zap.Logger {
	logger := NewByEnv(env, serviceName)
	zap.ReplaceGlobals(logger)
	return logger
}

// NewNop создаёт логгер который ничего не пишет (для тестов)
func NewNop() *zap.Logger {
	return zap.NewNop()
}

// NewTest создаёт логгер для тестов с записью в буфер
func NewTest() (*zap.Logger, *zapcore.BufferedWriteSyncer) {
	buf := &zapcore.BufferedWriteSyncer{WS: zapcore.AddSync(os.Stdout)}
	logger := New(
		WithLevel("debug"),
		WithEncoding("console"),
		WithOutput(buf),
		WithCaller(false),
	)
	return logger, buf
}
