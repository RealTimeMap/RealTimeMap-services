package redis

import (
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Address      string        `yaml:"address" env:"REDIS_ADDRESS"`
	DialTimeout  time.Duration `yaml:"dialTimeout" env:"REDIS_DIAL_TIMEOUT" env-default:"100ms"`
	ReadTimeout  time.Duration `yaml:"readTimeout" env:"REDIS_READ_TIMEOUT" env-default:"100ms"`
	WriteTimeout time.Duration `yaml:"writeTimeout" env:"REDIS_WRITE_TIMEOUT" env-default:"150ms"`
	MaxRetries   int           `yaml:"maxRetries" env:"REDIS_MAX_RETRIES" env-default:"1"`
	DB           int           `yaml:"db" env:"REDIS_DB" env-default:"0"`
}

func NewRedisCli(cfg Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		MaxRetries:   cfg.MaxRetries,
		DB:           cfg.DB,
	})
}
