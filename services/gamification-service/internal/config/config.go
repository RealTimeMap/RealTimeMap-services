package config

import (
	"time"

	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
)

type Database struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"db_name" env:"DB_NAME" env-required:"true"`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSL_MODE" env-default:"disable"`
}

type Kafka struct {
	Brokers        []string      `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	Topics         []string      `yaml:"topics"`
	GroupID        string        `yaml:"group_id" env:"KAFKA_GROUP_ID" env-default:"gamification-service"`
	MaxWait        time.Duration `yaml:"max_wait" env-default:"500ms"`
	CommitInterval time.Duration `yaml:"commit_interval" env-default:"0"`
}

type GRPC struct {
	Port int `yaml:"port" env:"GRPC_PORT" env-default:"50051"`
}

type HTTP struct {
	Port int `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
}
type Redis struct {
	Host string `yaml:"host" env:"REDIS_HOST" env-default:"localhost"`
}

type Config struct {
	Env           string   `env:"ENV" env-default:"local"`
	CacheStrategy string   ` yaml:"cacheStrategy" env:"CACHE_STRATEGY" env-required:"true"`
	Database      Database `yaml:"database"`
	Kafka         Kafka    `yaml:"kafka"`
	GRPC          GRPC     `yaml:"grpc"`
	HTTP          HTTP     `yaml:"http"`
	Redis         Redis    `yaml:"redis"`
}

func MustLoad() *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(
			"./config/config.yaml",
		),
	)
}
