package database

import "time"

type Options struct {
	MaxOpenConnections int
	MaxIdleConnections int
	ConnMaxLifetime    time.Duration
	ConnMaxIdleTime    time.Duration
	LogLevel           LogLevel
	SlowThreshold      time.Duration
}

type LogLevel int

const (
	LogSilent LogLevel = iota
	LogError
	LogWarn
	LogInfo
)

type Option func(*Options)

func defaultOptions() *Options {
	return &Options{
		MaxOpenConnections: 25,
		MaxIdleConnections: 5,
		ConnMaxLifetime:    5 * time.Minute,
		ConnMaxIdleTime:    5 * time.Minute,
		LogLevel:           LogError,
		SlowThreshold:      200 * time.Millisecond,
	}
}

func WithMaxOpenConnections(maxOpenConnections int) Option {
	return func(o *Options) {
		o.MaxOpenConnections = maxOpenConnections
	}
}

func WithMaxIdleConnections(maxIdleConnections int) Option {
	return func(o *Options) {
		o.MaxIdleConnections = maxIdleConnections
	}
}

func WithConnMaxLifetime(connMaxLifetime time.Duration) Option {
	return func(o *Options) {
		o.ConnMaxLifetime = connMaxLifetime
	}
}

func WithLogLevel(level LogLevel) Option {
	return func(o *Options) {
		o.LogLevel = level
	}
}
