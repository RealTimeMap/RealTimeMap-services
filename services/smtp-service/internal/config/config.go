package config

import (
	"time"

	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
)

// SMTP — параметры подключения к почтовому провайдеру.
type SMTP struct {
	Host     string `yaml:"host" env:"SMTP_HOST"`
	Port     int    `yaml:"port" env:"SMTP_PORT" env-default:"465"`
	UseSSL   bool   `yaml:"use_ssl" env:"SMTP_USE_SSL" env-default:"true"`
	Username string `yaml:"username" env:"SMTP_USERNAME"`
	Password string `yaml:"password" env:"SMTP_PASSWORD"`

	// From — адрес в заголовке письма. В MVP совпадает с Username,
	// при переезде на собственный домен (Yandex 360) разойдётся с ним.
	From string `yaml:"from" env:"SMTP_FROM"`

	// FromName — отображаемое имя отправителя.
	FromName string `yaml:"from_name" env:"SMTP_FROM_NAME" env-default:"RealTimeMap"`

	// AllowInsecure разрешает соединение без шифрования — только для
	// локального релея вроде MailHog. В продакшене письма несут персональные
	// данные, и открытый SMTP недопустим.
	AllowInsecure bool `yaml:"allow_insecure" env:"SMTP_ALLOW_INSECURE" env-default:"false"`

	// Timeout ограничивает установку соединения и SMTP-диалог.
	Timeout time.Duration `yaml:"timeout" env-default:"30s"`

	// IdleTimeout — после какого простоя соединение переоткрывается.
	// Серверы закрывают неактивные сессии молча.
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

// Database дублирует pkg/database.Config, добавляя теги env.
//
// Собственная структура, а не тип из pkg: тот размечен только yaml, и в
// контейнере креды из окружения были бы молча проигнорированы — сервис пошёл
// бы на localhost из файла. Тот же приём применён в gamification-service.
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

// Kafka — подписка на доменные события других сервисов.
type Kafka struct {
	Brokers []string `yaml:"brokers" env:"KAFKA_BROKERS" env-separator:"," env-default:"localhost:9092"`

	// Topics размечен env: список подписок меняется при добавлении события в
	// другом сервисе, и пересобирать образ ради одной строки не нужно.
	Topics []string `yaml:"topics" env:"KAFKA_TOPICS" env-separator:","`

	GroupID        string        `yaml:"group_id" env:"KAFKA_GROUP_ID" env-default:"smtp-service"`
	MaxWait        time.Duration `yaml:"max_wait" env-default:"500ms"`
	CommitInterval time.Duration `yaml:"commit_interval" env-default:"0"`
}

// Worker — параметры пула отправки.
type Worker struct {
	// Count — число воркеров. Оно же ограничивает количество одновременных
	// SMTP-соединений: провайдеры разрешают единицы, не десятки.
	Count int `yaml:"count" env:"WORKER_COUNT" env-default:"3"`

	// ClaimBatch — сколько писем воркер забирает за один заход в БД.
	ClaimBatch int `yaml:"claim_batch" env:"WORKER_CLAIM_BATCH" env-default:"10"`

	// PollInterval — пауза, когда очередь пуста.
	PollInterval time.Duration `yaml:"poll_interval" env-default:"2s"`

	// LockTimeout — на сколько письмо резервируется за воркером. По истечении
	// reaper вернёт его в очередь, считая воркер умершим.
	LockTimeout time.Duration `yaml:"lock_timeout" env-default:"2m"`

	// ReaperInterval — периодичность возврата зависших писем.
	ReaperInterval time.Duration `yaml:"reaper_interval" env-default:"1m"`

	// MaxAttempt — потолок попыток отправки одного письма.
	MaxAttempt uint `yaml:"max_attempt" env:"WORKER_MAX_ATTEMPT" env-default:"5"`

	// Backoff — задержки перед повторными попытками. Длина списка не обязана
	// совпадать с MaxAttempt: последнее значение используется для всех
	// последующих попыток. Джиттер добавляется на месте применения.
	Backoff []time.Duration `yaml:"backoff"`

	// DailyLimit — предохранитель от разгона при баге. 0 отключает проверку.
	// Ориентироваться на фактический лимит провайдера.
	DailyLimit int `yaml:"daily_limit" env:"WORKER_DAILY_LIMIT" env-default:"0"`

	// DomainRate — сколько писем в секунду допустимо слать на один домен
	// получателя. 0 отключает ограничение.
	DomainRate float64 `yaml:"domain_rate" env-default:"2"`

	// DomainBurst — запас писем, который можно отправить залпом, не упираясь
	// в DomainRate.
	DomainBurst int `yaml:"domain_burst" env-default:"5"`
}

// DefaultBackoff — задержки повторных попыток: 1m, 5m, 15m, 1h, 6h.
var DefaultBackoff = []time.Duration{
	time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	time.Hour,
	6 * time.Hour,
}

type Config struct {
	Env      string      `yaml:"env" env:"ENV" env-default:"local"`
	SMTP     SMTP        `yaml:"smtp"`
	Database Database    `yaml:"database"`
	Kafka    Kafka       `yaml:"kafka"`
	HTTP     http.Config `yaml:"http"`
	Worker   Worker      `yaml:"worker"`
}

func MustLoad() *Config {
	return MustLoadFrom("./config/config.yaml")
}

// MustLoadFrom загружает конфиг из указанного пути. Нужна тестам, которые
// запускаются не из корня сервиса и не могут положиться на относительный путь
// по умолчанию.
func MustLoadFrom(paths ...string) *Config {
	cfg := pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(paths...),
	)

	if cfg.SMTP.From == "" {
		cfg.SMTP.From = cfg.SMTP.Username
	}
	if len(cfg.Worker.Backoff) == 0 {
		cfg.Worker.Backoff = DefaultBackoff
	}

	return cfg
}
