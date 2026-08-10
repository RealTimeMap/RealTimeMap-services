package email

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventType — тип записи в истории письма.
type EventType string

const (
	EventCreated    EventType = "created"
	EventQueued     EventType = "queued"
	EventProcessing EventType = "processing"
	EventSent       EventType = "sent"
	EventFailed     EventType = "failed"
	EventRetried    EventType = "retried"
	EventCancelled  EventType = "cancelled"

	// EventBounced и EventComplained приходят асинхронно, уже после успешной
	// сдачи письма провайдеру, и в MVP не проставляются: SMTP Yandex не отдаёт
	// вебхуков. Появятся при переезде на SES/SendGrid/Postmark.
	EventBounced    EventType = "bounced"
	EventComplained EventType = "complained"
)

// Event — запись в истории письма. Append-only: строки не обновляются.
type Event struct {
	gorm.Model

	EmailID   uuid.UUID `gorm:"type:uuid;not null;index:idx_events_email_time,priority:1"`
	EventType EventType `gorm:"type:varchar(32);not null;index"`

	WorkerID string `gorm:"type:varchar(255)"`
	Provider string `gorm:"type:varchar(255)"`

	OccurredTime time.Time `gorm:"not null;index:idx_events_email_time,priority:2,sort:desc"`

	// TemplateVersion фиксирует, какая версия шаблона использовалась.
	// В MVP всегда 1; нужна, чтобы история отправки осталась воспроизводимой
	// после переезда шаблонов в БД (фаза 2), где версии меняются в рантайме.
	TemplateVersion uint

	// Details — произвольный контекст события (код ошибки SMTP, задержка retry).
	// Тело письма и данные шаблона сюда не пишутся: там персональные данные.
	Details map[string]any `gorm:"serializer:json;type:jsonb"`

	Attempt    uint
	MaxAttempt uint
}

func (Event) TableName() string {
	return "email_events"
}
