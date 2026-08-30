package key

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name       string
	Hint       string
	KeyHash    string `gorm:"uniqueIndex"`
	ExpiresAt  *time.Time
	RevokeAt   *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
}
