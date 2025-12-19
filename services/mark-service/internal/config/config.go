package config

import (
	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
)

// Grpc Конфиг для gRPC указывается НазваниеСервиса-Адресс смотреть example.config.yaml
type Grpc struct {
	UserService string `yaml:"user_service"`
}

type Database struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"db_name" env:"DB_NAME" env-required:"true"`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSL_MODE" env-default:"disable"`
}

// Kafka конфигурация для подключения к Kafka
type Kafka struct {
	Enabled       bool     `yaml:"enabled" env:"KAFKA_ENABLED" env-default:"false"`
	Brokers       []string `yaml:"brokers" env:"KAFKA_BROKERS" env-separator:","`
	ProducerTopic string   `yaml:"producerTopic" env:"KAFKA_PRODUCER_TOPIC" env-default:"mark-service.events"`
}

type Config struct {
	Env      string                `env:"ENV" env-default:"local"`
	Database Database              `yaml:"database"`
	Grpc     Grpc                  `yaml:"grpc"`
	Storage  storage.StorageConfig `yaml:"storage"`
	Kafka    Kafka                 `yaml:"kafka"`
}

func MustLoad() *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(
			"./config/config.yaml",
		),
	)
}
