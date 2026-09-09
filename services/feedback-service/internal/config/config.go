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

	// ServiceApiKey — ключ, по которому в сервис ходят другие сервисы
	// платформы (сейчас — таск-менеджер за перечнем багов и обратной
	// синхронизацией статуса).
	//
	// Пустое значение закрывает межсервисные маршруты: ненастроенный
	// ключ означает «эта интеграция здесь не поднята», а не «пускать всех».
	ServiceApiKey string `yaml:"service_api_key" env:"SERVICE_API_KEY"`
}

func MustLoad() *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(
			"./config/config.yaml",
		),
	)
}
