package config

import (
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/progress"
	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	servergrpc "github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/grpc"
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

type Config struct {
	Env          string                `env:"ENV" env-default:"local"`
	Database     Database              `yaml:"database"`
	Http         http.Config           `yaml:"http_server"`
	GRPC         servergrpc.Config     `yaml:"grpc"`
	Storage      storage.StorageConfig `yaml:"storage"`
	Gamification progress.Config       `yaml:"gamification"`
}

func MustLoad() *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(
			"./config/config.yaml",
		),
	)
}
