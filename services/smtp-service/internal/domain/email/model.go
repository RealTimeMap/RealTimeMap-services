package email

import (
	"time"

	"github.com/google/uuid"
)

// Status — состояние письма в очереди.
//
// Переходы:
//
//	queued -> processing -> sent
//	                     -> failed      (терминальная ошибка или исчерпаны попытки)
//	                     -> queued      (retryable-ошибка, отложено по backoff;
//	                                     либо возврат reaper-ом после смерти воркера)
//	queued -> cancelled                 (отмена до отправки)
type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusSent       Status = "sent"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// Email — письмо в очереди.
//
// Тело письма рендерится в момент постановки в очередь, а не при отправке:
// письмо неизменяемо с этого момента, ошибка рендера видна вызывающему
// синхронно, а повторная попытка гарантированно отправляет то же самое.
type Email struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	Status   Status `gorm:"type:varchar(32);not null;index:idx_emails_claim,priority:1;index:idx_emails_reap,priority:1"`
	Priority int    `gorm:"not null;default:0;index:idx_emails_claim,priority:3,sort:desc"`

	ToEmail string `gorm:"type:varchar(255);not null;index"`
	Subject string `gorm:"type:text;not null"`
	HTML    string `gorm:"type:text;not null"`

	TemplateID      string `gorm:"type:varchar(64);not null;index"`
	TemplateVersion uint   `gorm:"not null;default:1"`

	// DedupKey и DedupBucket вместе образуют уникальный ключ идемпотентности.
	// Bucket — номер временного окна, поэтому одинаковые письма отбрасываются
	// только внутри окна: легитимный повтор (юзер заново запросил код) проходит.
	DedupKey    string `gorm:"type:varchar(255);not null;uniqueIndex:idx_emails_dedup,priority:1"`
	DedupBucket int64  `gorm:"not null;uniqueIndex:idx_emails_dedup,priority:2"`

	// ScheduledAt — момент, начиная с которого письмо можно отправлять.
	// Двигается вперёд при retry и при откладывании по rate-limit.
	ScheduledAt time.Time `gorm:"not null;index:idx_emails_claim,priority:2"`

	// LockedUntil — до какого момента claim воркера считается действительным.
	// После истечения reaper возвращает письмо в очередь.
	LockedUntil *time.Time `gorm:"index:idx_emails_reap,priority:2"`
	WorkerID    string     `gorm:"type:varchar(64)"`

	Attempt    uint `gorm:"not null;default:0"`
	MaxAttempt uint `gorm:"not null;default:5"`

	LastError string `gorm:"type:text"`

	TraceID string `gorm:"type:varchar(64);index"`

	SentAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Email) TableName() string {
	return "emails"
}
