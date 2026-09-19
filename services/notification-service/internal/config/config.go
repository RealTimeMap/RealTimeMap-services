package config

import (
	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
)

// Database дублирует pkg/database.Config, добавляя теги env.
//
// Собственная структура, а не тип из pkg: тот размечен только yaml, и в
// контейнере креды из окружения были бы молча проигнорированы — сервис пошёл
// бы на localhost из файла. Тот же приём применён в smtp-service.
type Database struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"db_name" env:"DB_NAME" env-required:"true"`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSL_MODE" env-default:"disable"`
}

func (d Database) ToPkg() database.Config {
	return database.Config{
		Host:     d.Host,
		Port:     d.Port,
		User:     d.User,
		Password: d.Password,
		DBName:   d.DBName,
		SSLMode:  d.SSLMode,
	}
}

// Firebase — доступ к FCM. Key — публичный VAPID-ключ для web push, он же
// лежит в config.yaml. Приватная часть — в service account JSON, путь к
// которому задаётся GOOGLE_APPLICATION_CREDENTIALS и в git не коммитится.
type Firebase struct {
	ProjectID string `yaml:"project_id" env:"FCM_PROJECT_ID"`
	Key       string `yaml:"key" env:"FCM_KEY"`
}

type Config struct {
	Env      string      `yaml:"env" env:"ENV" env-default:"local"`
	Database Database    `yaml:"database"`
	HTTP     http.Config `yaml:"http"`
	Firebase Firebase    `yaml:"firebase"`
}

func MustLoad() *Config {
	return MustLoadFrom("./config/config.yaml")
}

// MustLoadFrom загружает конфиг из указанного пути. Нужна тестам, которые
// запускаются не из корня сервиса и не могут положиться на относительный путь
// по умолчанию.
func MustLoadFrom(paths ...string) *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(paths...),
	)
}
