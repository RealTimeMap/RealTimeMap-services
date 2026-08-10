package postgres

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Интеграционные тесты очереди: проверяют ровно то, что нельзя проверить
// моками — поведение Postgres. SKIP LOCKED при параллельном claim, отбрасывание
// дубля уникальным индексом, маппинг RETURNING * обратно в структуру.
//
// Требуют поднятой БД из config/config.yaml. Без неё пропускаются.
// Запуск из корня репозитория:
//
//	go test ./services/smtp-service/... -run Integration -v

func setupRepo(t *testing.T) (email.Repository, email.EventRepository, *gorm.DB, string) {
	t.Helper()

	cfg, err := loadConfig()
	if err != nil {
		t.Skipf("config not available: %v", err)
	}

	log := logger.NewNop()
	db, err := database.New(cfg.Database.ToPkg(), log)
	if err != nil {
		t.Skipf("database not available: %v", err)
	}

	if err := db.AutoMigrate(&email.Email{}, &email.Event{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Общий префикс на прогон: тесты идут по рабочей БД и должны убирать
	// за собой, не задевая чужие строки.
	run := uuid.New().String()[:8]
	t.Cleanup(func() {
		// Письма опознаются по префиксу ключа, а те, что дедуплицируются по
		// содержимому (ключ вида content:<hash>), — по адресу с тем же прогоном.
		const where = `dedup_key LIKE ? OR to_email = ? OR to_email LIKE ?`
		args := []any{"test:" + run + "%", "queue-test@example.com", "%" + run + "@example.com"}

		db.Exec(`DELETE FROM email_events WHERE email_id IN (SELECT id FROM emails WHERE `+where+`)`, args...)
		db.Exec(`DELETE FROM emails WHERE `+where, args...)
		database.Close(db)
	})

	return NewPgEmailRepository(db, log), NewPgEmailEventRepository(db, log), db, run
}

func loadConfig() (cfg *config.Config, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	// Тест запускается из своей директории, конфиг лежит в корне сервиса.
	return config.MustLoadFrom("../../../../config/config.yaml"), nil
}

func newEmail(run, key string, bucket int64, priority int, scheduledAt time.Time) *email.Email {
	return &email.Email{
		Status:      email.StatusQueued,
		Priority:    priority,
		ToEmail:     "queue-test@example.com",
		Subject:     "test",
		HTML:        "<p>test</p>",
		TemplateID:  "welcome",
		DedupKey:    fmt.Sprintf("test:%s:%s", run, key),
		DedupBucket: bucket,
		ScheduledAt: scheduledAt,
		MaxAttempt:  5,
	}
}

func TestIntegrationDedup(t *testing.T) {
	repo, _, _, run := setupRepo(t)
	ctx := context.Background()
	now := time.Now()

	first := newEmail(run, "dup", 1, 0, now)
	created, err := repo.Create(ctx, first)
	if err != nil || !created {
		t.Fatalf("first insert: created=%v err=%v", created, err)
	}

	second := newEmail(run, "dup", 1, 0, now)
	created, err = repo.Create(ctx, second)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if created {
		t.Error("duplicate was inserted, expected it to be dropped")
	}
	// Для вызывающего дубль — успех: он должен получить идентификатор уже
	// стоящего в очереди письма, а не ошибку.
	if second.ID != first.ID {
		t.Errorf("dedup returned id %s, want existing %s", second.ID, first.ID)
	}

	// Соседнее окно — уже другое письмо: иначе легитимный повтор (юзер заново
	// запросил код) никогда бы не прошёл.
	other := newEmail(run, "dup", 2, 0, now)
	created, err = repo.Create(ctx, other)
	if err != nil || !created {
		t.Errorf("different bucket: created=%v err=%v", created, err)
	}
}

func TestIntegrationClaim(t *testing.T) {
	repo, _, _, run := setupRepo(t)
	ctx := context.Background()
	now := time.Now()

	// Приоритетное письмо намеренно моложе: если сортировка по priority не
	// работает, порядок выдачи окажется обратным.
	high := newEmail(run, "high", 1, 10, now.Add(-time.Second))
	low := newEmail(run, "low", 1, 0, now.Add(-time.Minute))
	future := newEmail(run, "future", 1, 99, now.Add(time.Hour))

	for _, e := range []*email.Email{high, low, future} {
		if _, err := repo.Create(ctx, e); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	claimed, err := repo.Claim(ctx, "worker-"+run, 5, 2*time.Minute)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("claimed %d emails, want 2 (future one must be skipped)", len(claimed))
	}

	for _, e := range claimed {
		if e.ID == future.ID {
			t.Error("email scheduled in the future was claimed")
		}
	}

	got := claimed[0]
	if got.ID != high.ID {
		t.Errorf("first claimed is %s, want higher-priority %s", got.ID, high.ID)
	}
	if got.Status != email.StatusProcessing {
		t.Errorf("status = %s, want processing", got.Status)
	}
	if got.Attempt != 1 {
		t.Errorf("attempt = %d, want 1 (incremented before send)", got.Attempt)
	}
	if got.WorkerID != "worker-"+run {
		t.Errorf("worker_id = %q, want %q", got.WorkerID, "worker-"+run)
	}
	if got.LockedUntil == nil {
		t.Error("locked_until is nil, reaper would never release this row")
	}
	// RETURNING * должен восстановить всю строку, а не только изменённые поля:
	// воркер отправляет письмо прямо из результата claim.
	if got.Subject != "test" || got.HTML != "<p>test</p>" || got.TemplateID != "welcome" {
		t.Errorf("body not mapped from RETURNING: subject=%q html=%q template=%q",
			got.Subject, got.HTML, got.TemplateID)
	}
}

func TestIntegrationConcurrentClaim(t *testing.T) {
	repo, _, _, run := setupRepo(t)
	ctx := context.Background()
	now := time.Now()

	const total = 12
	for i := 0; i < total; i++ {
		if _, err := repo.Create(ctx, newEmail(run, fmt.Sprintf("conc-%d", i), 1, 0, now)); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	const workers = 4
	var mu sync.Mutex
	claimedBy := map[uuid.UUID][]string{}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			id := fmt.Sprintf("cw-%s-%d", run, w)
			got, err := repo.Claim(ctx, id, 5, time.Minute)
			if err != nil {
				t.Errorf("worker %d claim: %v", w, err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, e := range got {
				claimedBy[e.ID] = append(claimedBy[e.ID], id)
			}
		}(w)
	}
	wg.Wait()

	// Главное свойство очереди: одно письмо не уходит двум воркерам.
	for id, owners := range claimedBy {
		if len(owners) > 1 {
			t.Errorf("email %s claimed by %v — SKIP LOCKED did not isolate workers", id, owners)
		}
	}
}

func TestIntegrationTransitions(t *testing.T) {
	repo, _, _, run := setupRepo(t)
	ctx := context.Background()
	now := time.Now()

	e := newEmail(run, "transitions", 1, 0, now)
	if _, err := repo.Create(ctx, e); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := repo.Claim(ctx, "worker-"+run, 1, time.Minute); err != nil {
		t.Fatalf("claim: %v", err)
	}

	retryAt := now.Add(30 * time.Minute)
	if err := repo.Reschedule(ctx, e.ID, retryAt, "transient: timeout"); err != nil {
		t.Fatalf("reschedule: %v", err)
	}

	got, err := repo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != email.StatusQueued {
		t.Errorf("status = %s, want queued", got.Status)
	}
	if got.LockedUntil != nil {
		t.Error("locked_until must be cleared on reschedule, otherwise reaper races the retry")
	}
	// Счётчик попыток обязан пережить возврат в очередь — иначе потолок
	// max_attempt обходится бесконечно.
	if got.Attempt != 1 {
		t.Errorf("attempt = %d after reschedule, want 1", got.Attempt)
	}
	if !got.ScheduledAt.After(now.Add(20 * time.Minute)) {
		t.Errorf("scheduled_at = %v, want ~30m ahead", got.ScheduledAt)
	}

	if err := repo.MarkSent(ctx, e.ID, now); err != nil {
		t.Fatalf("mark sent: %v", err)
	}
	got, _ = repo.GetByID(ctx, e.ID)
	if got.Status != email.StatusSent || got.SentAt == nil {
		t.Errorf("status=%s sent_at=%v, want sent with timestamp", got.Status, got.SentAt)
	}

	failing := newEmail(run, "failing", 1, 0, now)
	if _, err := repo.Create(ctx, failing); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := repo.MarkFailed(ctx, failing.ID, "smtp 550: no such user"); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	got, _ = repo.GetByID(ctx, failing.ID)
	if got.Status != email.StatusFailed || got.LastError == "" {
		t.Errorf("status=%s last_error=%q, want failed with reason", got.Status, got.LastError)
	}
}

func TestIntegrationReleaseExpired(t *testing.T) {
	repo, _, db, run := setupRepo(t)
	ctx := context.Background()
	now := time.Now()
	past := now.Add(-10 * time.Minute)

	// Воркер умер, не дописав исход: claim протух.
	stuck := newEmail(run, "stuck", 1, 0, now)
	if _, err := repo.Create(ctx, stuck); err != nil {
		t.Fatalf("setup: %v", err)
	}
	db.Exec(`UPDATE emails SET status = ?, locked_until = ?, worker_id = ?, attempt = 1 WHERE id = ?`,
		email.StatusProcessing, past, "dead-worker", stuck.ID)

	// То же, но попытки исчерпаны — такое письмо возвращать в очередь нельзя.
	exhausted := newEmail(run, "exhausted", 1, 0, now)
	if _, err := repo.Create(ctx, exhausted); err != nil {
		t.Fatalf("setup: %v", err)
	}
	db.Exec(`UPDATE emails SET status = ?, locked_until = ?, attempt = 5, max_attempt = 5 WHERE id = ?`,
		email.StatusProcessing, past, exhausted.ID)

	released, failed, err := repo.ReleaseExpired(ctx, now)
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if released < 1 || failed < 1 {
		t.Errorf("released=%d failed=%d, want at least 1 of each", released, failed)
	}

	got, _ := repo.GetByID(ctx, stuck.ID)
	if got.Status != email.StatusQueued || got.WorkerID != "" || got.LockedUntil != nil {
		t.Errorf("stuck: status=%s worker=%q locked=%v, want clean queued",
			got.Status, got.WorkerID, got.LockedUntil)
	}

	got, _ = repo.GetByID(ctx, exhausted.ID)
	if got.Status != email.StatusFailed {
		t.Errorf("exhausted: status=%s, want failed", got.Status)
	}
}

func TestIntegrationQueries(t *testing.T) {
	repo, _, _, run := setupRepo(t)
	ctx := context.Background()
	now := time.Now()

	waiting := newEmail(run, "waiting", 1, 0, now.Add(-5*time.Minute))
	deferred := newEmail(run, "deferred", 1, 0, now.Add(2*time.Hour))
	for _, e := range []*email.Email{waiting, deferred} {
		if _, err := repo.Create(ctx, e); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	// Пустая очередь — самое обычное состояние, и запрос не должен на нём
	// падать: MIN() по пустой выборке возвращает NULL.
	empty, err := repo.OldestQueuedAge(ctx, now.Add(-100*time.Hour))
	if err != nil {
		t.Fatalf("empty queue age: %v", err)
	}
	if empty != 0 {
		t.Errorf("empty queue age = %v, want 0", empty)
	}

	// Возраст очереди меряется только по письмам, чьё время уже наступило:
	// отложенные ждут законно и сигналом затора не являются.
	age, err := repo.OldestQueuedAge(ctx, now)
	if err != nil {
		t.Fatalf("oldest age: %v", err)
	}
	if age < 4*time.Minute || age > 10*time.Minute {
		t.Errorf("age = %v, want ~5m (deferred email must not count)", age)
	}

	sent := newEmail(run, "sent", 1, 0, now)
	if _, err := repo.Create(ctx, sent); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := repo.MarkSent(ctx, sent.ID, now); err != nil {
		t.Fatalf("mark sent: %v", err)
	}

	count, err := repo.CountSentSince(ctx, now.Add(-time.Hour))
	if err != nil || count < 1 {
		t.Errorf("count sent: count=%d err=%v", count, err)
	}

	status := email.StatusQueued
	list, total, err := repo.List(ctx, email.Filter{
		Status:  &status,
		ToEmail: "queue-test@example.com",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total < 2 || len(list) < 2 {
		t.Errorf("list: total=%d len=%d, want at least 2 queued", total, len(list))
	}

	if _, err := repo.GetByID(ctx, uuid.New()); err == nil {
		t.Error("GetByID for unknown id returned no error")
	}
}

func TestIntegrationEvents(t *testing.T) {
	repo, events, _, run := setupRepo(t)
	ctx := context.Background()
	now := time.Now()

	e := newEmail(run, "events", 1, 0, now)
	if _, err := repo.Create(ctx, e); err != nil {
		t.Fatalf("setup: %v", err)
	}

	created := &email.Event{
		EmailID:    e.ID,
		EventType:  email.EventCreated,
		MaxAttempt: 5,
		Details:    map[string]any{"source": "integration-test"},
	}
	if err := events.Append(ctx, created); err != nil {
		t.Fatalf("append created: %v", err)
	}

	sentEvent := &email.Event{
		EmailID:      e.ID,
		EventType:    email.EventSent,
		Attempt:      1,
		MaxAttempt:   5,
		OccurredTime: now.Add(time.Second),
	}
	if err := events.Append(ctx, sentEvent); err != nil {
		t.Fatalf("append sent: %v", err)
	}

	list, err := events.ListByEmailID(ctx, e.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d events, want 2", len(list))
	}
	if list[0].EventType != email.EventCreated || list[1].EventType != email.EventSent {
		t.Errorf("events out of order: %s, %s", list[0].EventType, list[1].EventType)
	}
	if list[0].OccurredTime.IsZero() {
		t.Error("occurred_time was not defaulted on append")
	}
	if list[0].Details["source"] != "integration-test" {
		t.Errorf("jsonb details did not round-trip: %v", list[0].Details)
	}
}
