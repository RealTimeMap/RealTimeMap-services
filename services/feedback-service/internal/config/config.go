package config

import (
	pkgconfig "github.com/RealTimeMap/RealTimeMap-backend/pkg/config"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http"
)

// Kafka — шина событий. По ней gamification-service узнаёт о подтверждённых
// багах и награждает их авторов.
type Kafka struct {
	Enabled       bool     `yaml:"enabled" env:"KAFKA_ENABLED" env-default:"false"`
	Brokers       []string `yaml:"brokers" env:"KAFKA_BROKERS" env-separator:","`
	ProducerTopic string   `yaml:"producerTopic" env:"KAFKA_PRODUCER_TOPIC" env-default:"feedback-service.events"`

	// Topics — подписки консьюмера. Нужен только топик auth-сервиса: из него
	// приходит user.deleted. Пустой список — консьюмер не запускается.
	Topics  []string `yaml:"topics" env:"KAFKA_TOPICS" env-separator:","`
	GroupID string   `yaml:"group_id" env:"KAFKA_GROUP_ID" env-default:"feedback-service"`
}

type Config struct {
	Env      string          `env:"ENV" env-default:"local"`
	Http     http.Config     `yaml:"http"`
	Database database.Config `yaml:"database"`
	Kafka    Kafka           `yaml:"kafka"`

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
