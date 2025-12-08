package config

import (
	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
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

type Config struct {
	Env      string   `env:"ENV" env-default:"local"`
	Database Database `yaml:"database"`
	Grpc     Grpc     `yaml:"grpc"`
}

func MustLoad() *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(
			"./config/config.yaml",
		),
	)
}
