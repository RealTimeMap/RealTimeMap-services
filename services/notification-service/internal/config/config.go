package config

import (
	"time"

	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/redis"
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

type Firebase struct {
	ProjectID string `yaml:"project_id" env:"FCM_PROJECT_ID"`
	Key       string `yaml:"key" env:"FCM_KEY"`
}

type Kafka struct {
	Brokers []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092"`
	// Topics размечен env: список подписок меняется при добавлении события в
	// другом сервисе, и пересобирать образ ради одной строки не нужно.
	Topics         []string      `yaml:"topics" env:"KAFKA_TOPICS" env-separator:","`
	GroupID        string        `yaml:"group_id" env:"KAFKA_GROUP_ID" env-default:"notification-service"`
	MaxWait        time.Duration `yaml:"max_wait" env-default:"500ms"`
	CommitInterval time.Duration `yaml:"commit_interval" env-default:"0"`
}

// Collapse — параметры схлопывания серий уведомлений.
type Collapse struct {
	// Window — окно тишины после отправленного уведомления. Внутри него
	// события той же серии только считаются.
	//
	// Две минуты: короче — серия рвётся на несколько пушей при обычном темпе
	// переписки, длиннее — сводка приезжает, когда разговор уже закончился.
	Window time.Duration `yaml:"window" env:"COLLAPSE_WINDOW" env-default:"2m"`

	// SummaryInterval — как часто обход ищет закрытые серии, по которым надо
	// дослать сводку. Задаёт верхнюю границу задержки сводки поверх Window.
	SummaryInterval time.Duration `yaml:"summary_interval" env:"COLLAPSE_SUMMARY_INTERVAL" env-default:"30s"`

	// SummaryBatch — сколько серий обход разбирает за тик. Ограничение,
	// чтобы всплеск активности не превратил один тик в длинную рассылку.
	SummaryBatch int `yaml:"summary_batch" env:"COLLAPSE_SUMMARY_BATCH" env-default:"200"`
}

type Config struct {
	Env      string       `yaml:"env" env:"ENV" env-default:"local"`
	Database Database     `yaml:"database"`
	HTTP     http.Config  `yaml:"http"`
	Firebase Firebase     `yaml:"firebase"`
	Kafka    Kafka        `yaml:"kafka"`
	Redis    redis.Config `yaml:"redis"`
	Collapse Collapse     `yaml:"collapse"`
}

func MustLoad() *Config {
	return MustLoadFrom("./config/config.yaml")
}

func MustLoadFrom(paths ...string) *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(paths...),
	)
}
