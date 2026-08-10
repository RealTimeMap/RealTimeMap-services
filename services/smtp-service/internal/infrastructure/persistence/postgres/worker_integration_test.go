package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/worker"
	"github.com/google/uuid"
)

// Сквозная проверка пути «очередь → воркер → отправка» на настоящей БД.
// Юнит-тесты воркера работают с очередью в памяти; здесь проверяется, что
// claim, retry и reaper ведут себя так же поверх реального SKIP LOCKED.

type countingSender struct {
	mu      sync.Mutex
	sent    map[string]int
	results []error
	calls   int
}

func newCountingSender(results ...error) *countingSender {
	return &countingSender{sent: make(map[string]int), results: results}
}

func (s *countingSender) Send(_ context.Context, msg email.OutgoingMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.calls
	s.calls++

	if len(s.results) > 0 {
		if idx >= len(s.results) {
			idx = len(s.results) - 1
		}
		if err := s.results[idx]; err != nil {
			return err
		}
	}

	s.sent[msg.To]++
	return nil
}

func (s *countingSender) totalSent() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for _, n := range s.sent {
		total += n
	}
	return total
}

func (s *countingSender) deliveries(to string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sent[to]
}

func runWorkers(t *testing.T, pool *worker.Pool, until func() bool) {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- pool.Run() }()

	deadline := time.After(10 * time.Second)
	satisfied := false
	for !satisfied {
		select {
		case <-deadline:
			t.Error("workers did not reach the expected state in time")
			satisfied = true
		case <-time.After(20 * time.Millisecond):
			satisfied = until()
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Shutdown(ctx); err != nil {
		t.Errorf("shutdown: %v", err)
	}
	<-done
}

func newTestPool(repo email.Repository, events email.EventRepository, sender email.Sender, tune func(*worker.Config)) *worker.Pool {
	cfg := worker.Config{
		Count:        2,
		ClaimBatch:   5,
		PollInterval: 20 * time.Millisecond,
		LockTimeout:  time.Minute,
		Backoff:      []time.Duration{10 * time.Millisecond},
	}
	if tune != nil {
		tune(&cfg)
	}

	return worker.NewPool(cfg, repo, events, nil,
		func() (email.Sender, error) { return sender, nil },
		logger.NewNop())
}

func seedQueued(t *testing.T, repo email.Repository, run, key, to string) uuid.UUID {
	t.Helper()

	e := &email.Email{
		ID:          uuid.New(),
		Status:      email.StatusQueued,
		ToEmail:     to,
		Subject:     "Тема",
		HTML:        "<p>тело</p>",
		TemplateID:  "welcome",
		DedupKey:    fmt.Sprintf("test:%s:%s", run, key),
		DedupBucket: time.Now().Unix() / 300,
		ScheduledAt: time.Now().Add(-time.Second),
		MaxAttempt:  3,
	}

	if _, err := repo.Create(context.Background(), e); err != nil {
		t.Fatalf("seed %s: %v", key, err)
	}

	return e.ID
}

func TestIntegrationWorkerDeliversQueue(t *testing.T) {
	repo, events, _, run := setupRepo(t)
	ctx := context.Background()
	sender := newCountingSender()

	const total = 10
	ids := make([]uuid.UUID, 0, total)
	for i := 0; i < total; i++ {
		ids = append(ids, seedQueued(t, repo, run, fmt.Sprintf("bulk-%d", i), "queue-test@example.com"))
	}

	pool := newTestPool(repo, events, sender, nil)
	runWorkers(t, pool, func() bool {
		for _, id := range ids {
			e, err := repo.GetByID(ctx, id)
			if err != nil || e.Status != email.StatusSent {
				return false
			}
		}
		return true
	})

	// Каждое письмо отправляется ровно один раз, несмотря на двух воркеров,
	// работающих поверх одной очереди.
	if got := sender.totalSent(); got != total {
		t.Errorf("sent %d times for %d emails", got, total)
	}

	for _, id := range ids {
		e, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if e.SentAt == nil {
			t.Errorf("email %s has no sent_at", id)
		}
		if e.Attempt != 1 {
			t.Errorf("email %s attempt = %d, want 1", id, e.Attempt)
		}
	}
}

// 5xx не повторяется: попытки доставки на несуществующий адрес портят
// репутацию отправителя.
func TestIntegrationWorkerPermanentFailure(t *testing.T) {
	repo, events, _, run := setupRepo(t)
	ctx := context.Background()

	sender := newCountingSender(fmt.Errorf("%w: 550 no such user", email.ErrPermanent))
	id := seedQueued(t, repo, run, "permanent", "queue-test@example.com")

	pool := newTestPool(repo, events, sender, nil)
	runWorkers(t, pool, func() bool {
		e, err := repo.GetByID(ctx, id)
		return err == nil && e.Status == email.StatusFailed
	})

	e, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if e.Status != email.StatusFailed {
		t.Errorf("status = %s, want failed", e.Status)
	}
	if e.Attempt != 1 {
		t.Errorf("attempt = %d, want 1 (permanent errors are not retried)", e.Attempt)
	}
	if e.LastError == "" {
		t.Error("last_error is empty")
	}
}

// Временная ошибка возвращает письмо в очередь; после исчерпания попыток оно
// окончательно уходит в failed.
func TestIntegrationWorkerRetriesUntilExhausted(t *testing.T) {
	repo, events, _, run := setupRepo(t)
	ctx := context.Background()

	sender := newCountingSender(fmt.Errorf("%w: 451 busy", email.ErrTransient))
	id := seedQueued(t, repo, run, "retry", "queue-test@example.com")

	pool := newTestPool(repo, events, sender, func(c *worker.Config) {
		c.Count = 1
		c.Backoff = []time.Duration{5 * time.Millisecond}
	})

	runWorkers(t, pool, func() bool {
		e, err := repo.GetByID(ctx, id)
		return err == nil && e.Status == email.StatusFailed
	})

	e, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if e.Attempt != 3 {
		t.Errorf("attempt = %d, want exactly max_attempt (3)", e.Attempt)
	}

	// История переходов должна содержать и повторы, и итоговый провал.
	history, err := events.ListByEmailID(ctx, id)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	var retried, failed int
	for _, ev := range history {
		switch ev.EventType {
		case email.EventRetried:
			retried++
		case email.EventFailed:
			failed++
		}
	}
	if retried == 0 {
		t.Error("no retry events recorded")
	}
	if failed != 1 {
		t.Errorf("%d failure events recorded, want 1", failed)
	}
}

// Успех после временного отказа: письмо доходит, а не сгорает.
func TestIntegrationWorkerRecoversAfterTransientError(t *testing.T) {
	repo, events, _, run := setupRepo(t)
	ctx := context.Background()

	sender := newCountingSender(
		fmt.Errorf("%w: connection reset", email.ErrTransient),
		nil,
	)
	id := seedQueued(t, repo, run, "recover", "queue-test@example.com")

	pool := newTestPool(repo, events, sender, func(c *worker.Config) {
		c.Count = 1
		c.Backoff = []time.Duration{5 * time.Millisecond}
	})

	runWorkers(t, pool, func() bool {
		e, err := repo.GetByID(ctx, id)
		return err == nil && e.Status == email.StatusSent
	})

	e, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if e.Attempt != 2 {
		t.Errorf("attempt = %d, want 2", e.Attempt)
	}
	if sender.deliveries("queue-test@example.com") != 1 {
		t.Errorf("delivered %d times, want 1", sender.deliveries("queue-test@example.com"))
	}
}

// Письмо, отложенное на будущее, воркеры не трогают.
func TestIntegrationWorkerRespectsSchedule(t *testing.T) {
	repo, events, db, run := setupRepo(t)
	ctx := context.Background()

	sender := newCountingSender()
	id := seedQueued(t, repo, run, "future", "queue-test@example.com")
	db.Exec(`UPDATE emails SET scheduled_at = ? WHERE id = ?`, time.Now().Add(time.Hour), id)

	pool := newTestPool(repo, events, sender, nil)

	done := make(chan error, 1)
	go func() { done <- pool.Run() }()
	time.Sleep(300 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool.Shutdown(shutdownCtx)
	<-done

	if sender.totalSent() != 0 {
		t.Errorf("sent %d emails scheduled for the future", sender.totalSent())
	}

	e, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if e.Status != email.StatusQueued {
		t.Errorf("status = %s, want queued", e.Status)
	}
}

// Reaper возвращает письма, заблокированные умершим воркером: иначе они
// остались бы в processing навсегда, невидимые для claim.
func TestIntegrationReaperReleasesStuckEmails(t *testing.T) {
	repo, _, db, run := setupRepo(t)
	ctx := context.Background()

	stuck := seedQueued(t, repo, run, "stuck", "queue-test@example.com")
	db.Exec(`UPDATE emails SET status = ?, locked_until = ?, worker_id = ?, attempt = 1 WHERE id = ?`,
		email.StatusProcessing, time.Now().Add(-10*time.Minute), "dead-worker", stuck)

	reaper := worker.NewReaper(20*time.Millisecond, repo, logger.NewNop())

	done := make(chan error, 1)
	go func() { done <- reaper.Run() }()

	deadline := time.After(5 * time.Second)
	released := false
	for !released {
		select {
		case <-deadline:
			t.Fatal("reaper did not release the stuck email")
		case <-time.After(20 * time.Millisecond):
			e, err := repo.GetByID(ctx, stuck)
			released = err == nil && e.Status == email.StatusQueued
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := reaper.Shutdown(shutdownCtx); err != nil {
		t.Errorf("reaper shutdown: %v", err)
	}
	<-done

	e, err := repo.GetByID(ctx, stuck)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if e.WorkerID != "" || e.LockedUntil != nil {
		t.Errorf("released email still holds a claim: worker=%q locked=%v", e.WorkerID, e.LockedUntil)
	}
}

// Суточный потолок останавливает отправку: без него разогнавшийся цикл
// выберет квоту провайдера и приведёт к блокировке ящика.
func TestIntegrationWorkerHonoursDailyLimit(t *testing.T) {
	repo, events, db, run := setupRepo(t)
	ctx := context.Background()

	// Письмо, уже учтённое как отправленное сегодня.
	spent := seedQueued(t, repo, run, "already-sent", "queue-test@example.com")
	db.Exec(`UPDATE emails SET status = ?, sent_at = ? WHERE id = ?`,
		email.StatusSent, time.Now(), spent)

	pending := seedQueued(t, repo, run, "over-limit", "queue-test@example.com")

	sender := newCountingSender()
	pool := newTestPool(repo, events, sender, func(c *worker.Config) {
		c.DailyLimit = 1
	})

	done := make(chan error, 1)
	go func() { done <- pool.Run() }()
	time.Sleep(300 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool.Shutdown(shutdownCtx)
	<-done

	// Главное свойство: при исчерпанном лимите письмо не уходит адресату.
	if sender.totalSent() != 0 {
		t.Errorf("sent %d emails despite the daily limit", sender.totalSent())
	}

	e, err := repo.GetByID(ctx, pending)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	// Лимит ставит отправку на паузу, а не отменяет её: письмо должно остаться
	// живым и уйти, когда суточное окно сдвинется.
	//
	// Статус processing здесь тоже допустим: воркер циклически забирает письмо
	// и возвращает его в очередь, поэтому мгновенный снимок может застать его
	// захваченным. Недопустимы только терминальные состояния.
	switch e.Status {
	case email.StatusQueued, email.StatusProcessing:
	default:
		t.Errorf("status = %s, want the email to stay pending (limit pauses delivery, not fails it)", e.Status)
	}
}

// Остановка не должна терять письма: забранные, но не отправленные
// возвращаются в очередь сразу, а не ждут истечения LockTimeout.
func TestIntegrationWorkerReleasesOnShutdown(t *testing.T) {
	repo, events, _, run := setupRepo(t)
	ctx := context.Background()

	const total = 6
	ids := make([]uuid.UUID, 0, total)
	for i := 0; i < total; i++ {
		ids = append(ids, seedQueued(t, repo, run, fmt.Sprintf("shutdown-%d", i), "queue-test@example.com"))
	}

	// Отправка нарочито медленная, чтобы остановка застала батч в работе.
	slow := &slowSender{delay: 200 * time.Millisecond}
	pool := newTestPool(repo, events, slow, func(c *worker.Config) {
		c.Count = 1
		c.ClaimBatch = total
	})

	done := make(chan error, 1)
	go func() { done <- pool.Run() }()
	time.Sleep(250 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Shutdown(shutdownCtx); err != nil {
		t.Errorf("shutdown: %v", err)
	}
	<-done

	stillProcessing := 0
	for _, id := range ids {
		e, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if e.Status == email.StatusProcessing {
			stillProcessing++
		}
	}

	if stillProcessing > 0 {
		t.Errorf("%d emails left stuck in processing after shutdown", stillProcessing)
	}
}

type slowSender struct {
	delay time.Duration
	mu    sync.Mutex
	calls int
}

func (s *slowSender) Send(ctx context.Context, _ email.OutgoingMessage) error {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()

	select {
	case <-time.After(s.delay):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: %v", email.ErrTransient, errors.New("interrupted"))
	}
}
