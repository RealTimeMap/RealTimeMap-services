package postgres

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PgEmailRepository struct {
	db *gorm.DB

	logger *zap.Logger
}

func NewPgEmailRepository(db *gorm.DB, logger *zap.Logger) email.Repository {
	return &PgEmailRepository{
		db:     db,
		logger: logger,
	}
}

// Create ставит письмо в очередь, отбрасывая дубликат по паре
// (dedup_key, dedup_bucket).
//
// ON CONFLICT DO NOTHING вместо предварительного SELECT: проверка «есть ли
// уже такое» и вставка в разных запросах — это гонка, две параллельные
// доставки одного Kafka-события прошли бы обе.
func (r *PgEmailRepository) Create(ctx context.Context, e *email.Email) (bool, error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}

	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dedup_key"}, {Name: "dedup_bucket"}},
			DoNothing: true,
		}).
		Create(e)

	if res.Error != nil {
		return false, res.Error
	}

	if res.RowsAffected > 0 {
		return true, nil
	}

	// Конфликт: письмо уже стоит в очереди. Возвращаем существующее, чтобы
	// вызывающий получил его идентификатор — для него дубль это успех
	// («письмо уже принято»), а не ошибка.
	existing, err := r.getByDedup(ctx, e.DedupKey, e.DedupBucket)
	if err != nil {
		return false, err
	}
	*e = *existing

	return false, nil
}

func (r *PgEmailRepository) getByDedup(ctx context.Context, key string, bucket int64) (*email.Email, error) {
	var found email.Email
	err := r.db.WithContext(ctx).
		Where("dedup_key = ? AND dedup_bucket = ?", key, bucket).
		Take(&found).Error
	if err != nil {
		return nil, err
	}
	return &found, nil
}

func (r *PgEmailRepository) GetByID(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	var found email.Email

	err := r.db.WithContext(ctx).Take(&found, "id = ?", id).Error
	if err != nil {
		// Различать «нет такого письма» и «БД недоступна» обязательно:
		// иначе падение Postgres выглядит как отсутствие записи.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, email.ErrNotFound(id.String())
		}
		return nil, err
	}

	return &found, nil
}

func (r *PgEmailRepository) List(ctx context.Context, f email.Filter) ([]email.Email, int64, error) {
	q := r.db.WithContext(ctx).Model(&email.Email{})

	if f.Status != nil && *f.Status != "" {
		q = q.Where("status = ?", *f.Status)
	}
	if f.ToEmail != "" {
		q = q.Where("to_email = ?", f.ToEmail)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	var records []email.Email
	err := q.Limit(limit).
		Offset(f.Offset).
		Order("created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// claimQuery резервирует письма за воркером одним запросом.
//
// SKIP LOCKED пропускает строки, уже захваченные другим воркером, вместо
// ожидания их разблокировки — параллельные воркеры разбирают очередь, не
// толкаясь. Внутренний SELECT ... FOR UPDATE нужен, чтобы между выбором
// строк и их обновлением никто не вклинился.
//
// attempt инкрементируется здесь же, до отправки: если воркер умрёт на
// SMTP-диалоге, попытка уже посчитана и потолок нельзя обойти бесконечно.
const claimQuery = `
UPDATE emails SET
    status       = @processing,
    worker_id    = @worker_id,
    locked_until = @locked_until,
    attempt      = attempt + 1,
    updated_at   = @now
WHERE id IN (
    SELECT id FROM emails
    WHERE status = @queued
      AND scheduled_at <= @now
    ORDER BY priority DESC, scheduled_at
    FOR UPDATE SKIP LOCKED
    LIMIT @limit
)
RETURNING *`

func (r *PgEmailRepository) Claim(ctx context.Context, workerID string, limit int, lockFor time.Duration) ([]email.Email, error) {
	now := time.Now()

	var claimed []email.Email
	err := r.db.WithContext(ctx).Raw(claimQuery, map[string]any{
		"processing":   email.StatusProcessing,
		"queued":       email.StatusQueued,
		"worker_id":    workerID,
		"locked_until": now.Add(lockFor),
		"now":          now,
		"limit":        limit,
	}).Scan(&claimed).Error

	if err != nil {
		return nil, err
	}

	// ORDER BY во внутреннем SELECT решает, какие письма забрать, но не задаёт
	// порядок строк в RETURNING — Postgres отдаёт их в порядке обновления.
	// Воркер обрабатывает батч последовательно, поэтому порядок восстанавливаем:
	// иначе срочное письмо уходит после накопившихся обычных.
	sort.Slice(claimed, func(i, j int) bool {
		if claimed[i].Priority != claimed[j].Priority {
			return claimed[i].Priority > claimed[j].Priority
		}
		return claimed[i].ScheduledAt.Before(claimed[j].ScheduledAt)
	})

	return claimed, nil
}

func (r *PgEmailRepository) MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&email.Email{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       email.StatusSent,
			"sent_at":      sentAt,
			"locked_until": nil,
			"last_error":   "",
			"updated_at":   time.Now(),
		}).Error
}

func (r *PgEmailRepository) MarkFailed(ctx context.Context, id uuid.UUID, reason string) error {
	return r.db.WithContext(ctx).
		Model(&email.Email{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       email.StatusFailed,
			"last_error":   reason,
			"locked_until": nil,
			"updated_at":   time.Now(),
		}).Error
}

func (r *PgEmailRepository) Reschedule(ctx context.Context, id uuid.UUID, at time.Time, reason string) error {
	return r.db.WithContext(ctx).
		Model(&email.Email{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       email.StatusQueued,
			"scheduled_at": at,
			"locked_until": nil,
			"worker_id":    "",
			"last_error":   reason,
			"updated_at":   time.Now(),
		}).Error
}

// ReleaseExpired возвращает в очередь письма, чей claim истёк.
//
// Письмо попадает сюда, если воркер умер после захвата. Есть шанс, что он
// успел сдать письмо провайдеру, но не успел записать sent — тогда повторная
// отправка даст дубль. Это принято сознательно: SMTP и Postgres не в одной
// транзакции, и неполученное письмо хуже полученного дважды.
func (r *PgEmailRepository) ReleaseExpired(ctx context.Context, now time.Time) (int64, int64, error) {
	// Исчерпавшие попытки — в failed, иначе они возвращались бы в очередь вечно.
	failedRes := r.db.WithContext(ctx).
		Model(&email.Email{}).
		Where("status = ? AND locked_until < ? AND attempt >= max_attempt",
			email.StatusProcessing, now).
		Updates(map[string]any{
			"status":       email.StatusFailed,
			"last_error":   "max attempts exceeded after worker timeout",
			"locked_until": nil,
			"updated_at":   now,
		})
	if failedRes.Error != nil {
		return 0, 0, failedRes.Error
	}

	releasedRes := r.db.WithContext(ctx).
		Model(&email.Email{}).
		Where("status = ? AND locked_until < ? AND attempt < max_attempt",
			email.StatusProcessing, now).
		Updates(map[string]any{
			"status":       email.StatusQueued,
			"worker_id":    "",
			"locked_until": nil,
			"last_error":   "released after worker timeout",
			"updated_at":   now,
		})
	if releasedRes.Error != nil {
		return 0, 0, releasedRes.Error
	}

	return releasedRes.RowsAffected, failedRes.RowsAffected, nil
}

func (r *PgEmailRepository) CountSentSince(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&email.Email{}).
		Where("status = ? AND sent_at >= ?", email.StatusSent, since).
		Count(&count).Error
	return count, err
}

// OldestQueuedAge — как долго ждёт самое старое письмо, готовое к отправке.
//
// Письма, отложенные на будущее (retry, rate-limit), не учитываются: они ждут
// законно. Считаются только те, чьё время уже наступило — их возраст и есть
// мера того, справляется ли пул воркеров.
func (r *PgEmailRepository) OldestQueuedAge(ctx context.Context, now time.Time) (time.Duration, error) {
	// sql.NullTime, а не *time.Time: на пустой очереди MIN() возвращает NULL,
	// и сканирование в указатель падает с ошибкой драйвера. Пустая очередь —
	// самое обычное состояние, и health-проверка не должна на нём спотыкаться.
	var oldest sql.NullTime

	err := r.db.WithContext(ctx).
		Model(&email.Email{}).
		Select("MIN(scheduled_at)").
		Where("status = ? AND scheduled_at <= ?", email.StatusQueued, now).
		Scan(&oldest).Error
	if err != nil {
		return 0, err
	}

	if !oldest.Valid {
		return 0, nil
	}

	return now.Sub(oldest.Time), nil
}
