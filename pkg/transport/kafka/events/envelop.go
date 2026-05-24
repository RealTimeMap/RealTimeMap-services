package events

import (
	"time"

	"github.com/google/uuid"
)

type Envelop struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}

func NewEnvelop(eventType string) Envelop {
	return Envelop{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
	}
}
