package email

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Filter — параметры выборки писем для админки.
type Filter struct {
	Status  *Status
	ToEmail string
	From    *time.Time
	To      *time.Time
	Limit   int
	Offset  int
}

// Repository — доступ к очереди писем.
type Repository interface {
	// Create ставит письмо в очередь. Если письмо с такой парой
	// (DedupKey, DedupBucket) уже есть, возвращает существующую запись и
	// created == false, не создавая дубликата.
	Create(ctx context.Context, e *Email) (created bool, err error)

	GetByID(ctx context.Context, id uuid.UUID) (*Email, error)
	List(ctx context.Context, f Filter) ([]Email, int64, error)

	// Claim атомарно резервирует до limit писем, готовых к отправке, помечая
	// их processing и инкрементируя Attempt.
	//
	// Транзакция обязана быть короткой: отправка происходит уже вне её.
	// Держать транзакцию открытой на время SMTP-диалога — это блокировка на
	// секунды, раздутый xmin horizon и неработающий autovacuum.
	Claim(ctx context.Context, workerID string, limit int, lockFor time.Duration) ([]Email, error)

	MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string) error

	// Reschedule возвращает письмо в очередь с отложенным временем отправки.
	// Используется и для retry после временной ошибки, и для откладывания
	// по rate-limit.
	Reschedule(ctx context.Context, id uuid.UUID, at time.Time, reason string) error

	// ReleaseExpired возвращает в очередь письма, чей claim истёк: воркер,
	// забравший их, не подал признаков жизни. Письма, исчерпавшие попытки,
	// переводятся в failed.
	ReleaseExpired(ctx context.Context, now time.Time) (released int64, failed int64, err error)

	// CountSentSince — счётчик отправленного за период. Предохранитель от
	// разгона при баге, сверяется с суточным лимитом провайдера.
	CountSentSince(ctx context.Context, since time.Time) (int64, error)

	// OldestQueuedAge — возраст самого старого письма, ожидающего отправки.
	// Единственный симптом вставшей очереди, видимый до жалоб пользователей:
	// письмо не имеет обратной связи в UI. Возвращает 0, если очередь пуста.
	OldestQueuedAge(ctx context.Context, now time.Time) (time.Duration, error)
}

// EventRepository — append-only история переходов письма.
type EventRepository interface {
	Append(ctx context.Context, e *Event) error
	ListByEmailID(ctx context.Context, emailID uuid.UUID) ([]Event, error)
}
