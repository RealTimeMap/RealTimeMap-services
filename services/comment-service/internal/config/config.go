package config

import (
	"time"

	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
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
	Enabled       bool     `yaml:"enabled" env:"KAFKA_ENABLED" env-default:"false"`
	Brokers       []string `yaml:"brokers" env:"KAFKA_BROKERS" env-separator:","`
	ProducerTopic string   `yaml:"producerTopic" env:"KAFKA_PRODUCER_TOPIC" env-default:"comment-service.events"`
}

type ProfileGRPC struct {
	Address string        `yaml:"address" env:"PROFILE_GRPC_ADDRESS" env-default:"profile-service:9090"`
	Timeout time.Duration `yaml:"timeout" env:"PROFILE_GRPC_TIMEOUT" env-default:"3s"`
}

type Config struct {
	Env      string      `yaml:"env" env:"APP_ENV"`
	Database Database    `yaml:"database"`
	HTTP     http.Config `yaml:"http"`
	Kafka    Kafka       `yaml:"kafka"`
	Profile  ProfileGRPC `yaml:"profile"`
}

func MustLoad() *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(
			"./config/config.yaml",
		),
	)
}
