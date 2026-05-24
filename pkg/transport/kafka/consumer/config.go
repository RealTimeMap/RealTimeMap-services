package consumer

import "time"

// Config конфиг для kafka consumer.
type Config struct {
	Brokers        []string
	Topics         []string // Несколько топиков для подписки
	GroupID        string
	MinBytes       int
	MaxBytes       int
	MaxWait        time.Duration
	CommitInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 0, // ручной коммит
	}
}

// WithBrokers устанавливает адреса брокеров.
func (c Config) WithBrokers(brokers ...string) Config {
	c.Brokers = brokers
	return c
}

// WithTopics устанавливает список топиков для подписки.
func (c Config) WithTopics(topics ...string) Config {
	c.Topics = topics
	return c
}

// WithTopic устанавливает один топик (для обратной совместимости).
func (c Config) WithTopic(topic string) Config {
	c.Topics = []string{topic}
	return c
}

// WithGroupID устанавливает group ID.
func (c Config) WithGroupID(groupID string) Config {
	c.GroupID = groupID
	return c
}
