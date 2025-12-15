package consumer

import "time"

// Config конфиг для kafka

type Config struct {
	Brokers        []string
	Topic          string // Топик
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

// WithTopic устанавливает топик.
func (c Config) WithTopic(topic string) Config {
	c.Topic = topic
	return c
}

// WithGroupID устанавливает group ID.
func (c Config) WithGroupID(groupID string) Config {
	c.GroupID = groupID
	return c
}
