package config

import (
	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
)

type Config struct {
	Env      string          `env:"ENV" env-default:"local"`
	Http     http.Config     `yaml:"http"`
	Database database.Config `yaml:"database"`
}

func MustLoad() *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(
			"./config/config.yaml",
		),
	)
}
