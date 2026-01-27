package config

import pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"

type Database struct {
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port     int    `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"db_name" env:"DB_NAME" env-required:"true"`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSL_MODE" env-default:"disable"`
}

type HTTP struct {
	Port int `yaml:"port" env:"HTTP_PORT" env-default:"8080"`
}

type Kafka struct {
	Enabled       bool     `yaml:"enabled" env:"KAFKA_ENABLED" env-default:"false"`
	Brokers       []string `yaml:"brokers" env:"KAFKA_BROKERS" env-separator:","`
	ProducerTopic string   `yaml:"producerTopic" env:"KAFKA_PRODUCER_TOPIC" env-default:"comment-service.events"`
}

type Config struct {
	Env      string   `yaml:"env" env:"APP_ENV"`
	Database Database `yaml:"database"`
	HTTP     HTTP     `yaml:"http"`
	Kafka    Kafka    `yaml:"kafka"`
}

func MustLoad() *Config {
	return pkgconfig.MustLoad[Config](
		pkgconfig.WithPaths(
			"./config/config.yaml",
		),
	)
}
