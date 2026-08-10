package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/google/uuid"
)

// queueRepo — очередь в памяти. Воспроизводит поведение, от которого зависит
// воркер: claim выдаёт письмо один раз и инкрементирует attempt.
type queueRepo struct {
	mu     sync.Mutex
	emails map[uuid.UUID]*email.Email
	order  []uuid.UUID

	claimErr  error
	sentCount int64
}

func newQueueRepo() *queueRepo {
	return &queueRepo{emails: make(map[uuid.UUID]*email.Email)}
}

func (r *queueRepo) add(e *email.Email) *email.Email {
	r.mu.Lock()
	defer r.mu.Unlock()

	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.MaxAttempt == 0 {
		e.MaxAttempt = 3
	}
	if e.Status == "" {
		e.Status = email.StatusQueued
	}
	stored := *e
	r.emails[e.ID] = &stored
	r.order = append(r.order, e.ID)

	return &stored
}

func (r *queueRepo) snapshot(id uuid.UUID) email.Email {
	r.mu.Lock()
	defer r.mu.Unlock()
	return *r.emails[id]
}

func (r *queueRepo) Claim(_ context.Context, workerID string, limit int, lockFor time.Duration) ([]email.Email, error) {
	if r.claimErr != nil {
		return nil, r.claimErr
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	var claimed []email.Email

	for _, id := range r.order {
		if len(claimed) >= limit {
			break
		}
		e := r.emails[id]
		if e.Status != email.StatusQueued || e.ScheduledAt.After(now) {
			continue
		}

		lockedUntil := now.Add(lockFor)
		e.Status = email.StatusProcessing
		e.WorkerID = workerID
		e.LockedUntil = &lockedUntil
		e.Attempt++

		claimed = append(claimed, *e)
	}

	return claimed, nil
}

func (r *queueRepo) MarkSent(_ context.Context, id uuid.UUID, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.emails[id]
	e.Status = email.StatusSent
	e.SentAt = &at
	e.LockedUntil = nil
	r.sentCount++

	return nil
}

func (r *queueRepo) MarkFailed(_ context.Context, id uuid.UUID, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.emails[id]
	e.Status = email.StatusFailed
	e.LastError = reason
	e.LockedUntil = nil

	return nil
}

func (r *queueRepo) Reschedule(_ context.Context, id uuid.UUID, at time.Time, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e := r.emails[id]
	e.Status = email.StatusQueued
	e.ScheduledAt = at
	e.LastError = reason
	e.LockedUntil = nil
	e.WorkerID = ""

	return nil
}

func (r *queueRepo) CountSentSince(context.Context, time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sentCount, nil
}

func (r *queueRepo) Create(context.Context, *email.Email) (bool, error) { return true, nil }
func (r *queueRepo) GetByID(_ context.Context, id uuid.UUID) (*email.Email, error) {
	e := r.snapshot(id)
	return &e, nil
}
func (r *queueRepo) List(context.Context, email.Filter) ([]email.Email, int64, error) {
	return nil, 0, nil
}
func (r *queueRepo) ReleaseExpired(context.Context, time.Time) (int64, int64, error) {
	return 0, 0, nil
}
func (r *queueRepo) OldestQueuedAge(context.Context, time.Time) (time.Duration, error) {
	return 0, nil
}

type recordingEvents struct {
	mu     sync.Mutex
	events []email.Event
}

func (e *recordingEvents) Append(_ context.Context, ev *email.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, *ev)
	return nil
}

func (e *recordingEvents) ListByEmailID(context.Context, uuid.UUID) ([]email.Event, error) {
	return nil, nil
}

func (e *recordingEvents) types() []email.EventType {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]email.EventType, 0, len(e.events))
	for _, ev := range e.events {
		out = append(out, ev.EventType)
	}
	return out
}

// scriptedSender отвечает по заранее заданному сценарию и считает попытки.
type scriptedSender struct {
	mu       sync.Mutex
	attempts int
	results  []error // по одной записи на попытку; после конца — последняя
	sentTo   []string
}

func (s *scriptedSender) Send(_ context.Context, msg email.OutgoingMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.attempts
	s.attempts++

	if len(s.results) == 0 {
		s.sentTo = append(s.sentTo, msg.To)
		return nil
	}
	if idx >= len(s.results) {
		idx = len(s.results) - 1
	}

	if s.results[idx] == nil {
		s.sentTo = append(s.sentTo, msg.To)
	}
	return s.results[idx]
}

func (s *scriptedSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempts
}

type fixedLimiter struct{ delay time.Duration }

func (f fixedLimiter) Reserve(string) time.Duration { return f.delay }

func queuedEmail() *email.Email {
	return &email.Email{
		Status:      email.StatusQueued,
		ToEmail:     "user@example.com",
		Subject:     "Тема",
		HTML:        "<p>тело</p>",
		TemplateID:  "welcome",
		ScheduledAt: time.Now().Add(-time.Second),
		MaxAttempt:  3,
	}
}

// runPool гоняет пул, пока не выполнится условие, затем останавливает его.
func runPool(t *testing.T, p *Pool, until func() bool) {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- p.Run() }()

	deadline := time.After(3 * time.Second)
	for !until() {
		select {
		case <-deadline:
			t.Error("condition not met in time")
			goto stop
		case <-time.After(5 * time.Millisecond):
		}
	}

stop:
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Shutdown(ctx); err != nil {
		t.Errorf("shutdown: %v", err)
	}
	<-done
}

func newPool(repo *queueRepo, events *recordingEvents, sender email.Sender, tune func(*Config)) *Pool {
	cfg := Config{
		Count:        1,
		ClaimBatch:   10,
		PollInterval: 10 * time.Millisecond,
		LockTimeout:  time.Minute,
		Backoff:      []time.Duration{50 * time.Millisecond, 100 * time.Millisecond},
	}
	if tune != nil {
		tune(&cfg)
	}

	return NewPool(cfg, repo, events, nil,
		func() (email.Sender, error) { return sender, nil },
		logger.NewNop())
}

func TestWorkerSendsQueuedEmail(t *testing.T) {
	repo := newQueueRepo()
	events := &recordingEvents{}
	sender := &scriptedSender{}

	e := repo.add(queuedEmail())
	pool := newPool(repo, events, sender, nil)

	runPool(t, pool, func() bool {
		return repo.snapshot(e.ID).Status == email.StatusSent
	})

	stored := repo.snapshot(e.ID)
	if stored.Status != email.StatusSent {
		t.Errorf("status = %s, want sent", stored.Status)
	}
	if stored.SentAt == nil {
		t.Error("sent_at not set")
	}
	if sender.count() != 1 {
		t.Errorf("sent %d times, want 1", sender.count())
	}

	var sawSent bool
	for _, tp := range events.types() {
		if tp == email.EventSent {
			sawSent = true
		}
	}
	if !sawSent {
		t.Error("sent event not recorded")
	}
}

// 5xx означает «адреса не существует»: повторные попытки на такие адреса
// портят репутацию отправителя, поэтому письмо сгорает сразу.
func TestWorkerPermanentErrorIsNotRetried(t *testing.T) {
	repo := newQueueRepo()
	sender := &scriptedSender{results: []error{
		fmt.Errorf("%w: 550 no such user", email.ErrPermanent),
	}}

	e := repo.add(queuedEmail())
	pool := newPool(repo, &recordingEvents{}, sender, nil)

	runPool(t, pool, func() bool {
		return repo.snapshot(e.ID).Status == email.StatusFailed
	})

	stored := repo.snapshot(e.ID)
	if stored.Status != email.StatusFailed {
		t.Errorf("status = %s, want failed", stored.Status)
	}
	if stored.Attempt != 1 {
		t.Errorf("attempt = %d, want 1 (no retries for permanent errors)", stored.Attempt)
	}
	if sender.count() != 1 {
		t.Errorf("sender called %d times, want 1", sender.count())
	}
}

// Временная ошибка возвращает письмо в очередь с отложенным временем.
func TestWorkerTransientErrorSchedulesRetry(t *testing.T) {
	repo := newQueueRepo()
	sender := &scriptedSender{results: []error{
		fmt.Errorf("%w: 451 mailbox busy", email.ErrTransient),
	}}

	e := repo.add(queuedEmail())
	pool := newPool(repo, &recordingEvents{}, sender, func(c *Config) {
		// Крупная задержка, чтобы письмо не подхватили повторно во время теста.
		c.Backoff = []time.Duration{time.Hour}
	})

	runPool(t, pool, func() bool {
		s := repo.snapshot(e.ID)
		return s.Status == email.StatusQueued && s.Attempt == 1
	})

	stored := repo.snapshot(e.ID)
	if stored.Status != email.StatusQueued {
		t.Errorf("status = %s, want queued", stored.Status)
	}
	if !stored.ScheduledAt.After(time.Now().Add(30 * time.Minute)) {
		t.Errorf("scheduled_at = %v, want ~1h ahead", stored.ScheduledAt)
	}
	// Счётчик попыток обязан пережить возврат: иначе потолок не достигается.
	if stored.Attempt != 1 {
		t.Errorf("attempt = %d, want 1", stored.Attempt)
	}
}

// После исчерпания попыток письмо перестаёт возвращаться в очередь.
func TestWorkerStopsAfterMaxAttempts(t *testing.T) {
	repo := newQueueRepo()
	sender := &scriptedSender{results: []error{
		fmt.Errorf("%w: timeout", email.ErrTransient),
	}}

	seed := queuedEmail()
	seed.MaxAttempt = 3
	e := repo.add(seed)

	pool := newPool(repo, &recordingEvents{}, sender, func(c *Config) {
		c.Backoff = []time.Duration{time.Millisecond}
	})

	runPool(t, pool, func() bool {
		return repo.snapshot(e.ID).Status == email.StatusFailed
	})

	stored := repo.snapshot(e.ID)
	if stored.Status != email.StatusFailed {
		t.Errorf("status = %s, want failed", stored.Status)
	}
	if stored.Attempt != 3 {
		t.Errorf("attempt = %d, want exactly max_attempt (3)", stored.Attempt)
	}
}

// Успех после временной ошибки: письмо уходит, статус — sent.
func TestWorkerRecoversAfterTransientError(t *testing.T) {
	repo := newQueueRepo()
	sender := &scriptedSender{results: []error{
		fmt.Errorf("%w: temporary", email.ErrTransient),
		nil,
	}}

	e := repo.add(queuedEmail())
	pool := newPool(repo, &recordingEvents{}, sender, func(c *Config) {
		c.Backoff = []time.Duration{time.Millisecond}
	})

	runPool(t, pool, func() bool {
		return repo.snapshot(e.ID).Status == email.StatusSent
	})

	if got := repo.snapshot(e.ID).Attempt; got != 2 {
		t.Errorf("attempt = %d, want 2", got)
	}
}

// Отложенное письмо не должно отправляться раньше срока.
func TestWorkerSkipsScheduledEmails(t *testing.T) {
	repo := newQueueRepo()
	sender := &scriptedSender{}

	seed := queuedEmail()
	seed.ScheduledAt = time.Now().Add(time.Hour)
	e := repo.add(seed)

	pool := newPool(repo, &recordingEvents{}, sender, nil)

	done := make(chan error, 1)
	go func() { done <- pool.Run() }()
	time.Sleep(150 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool.Shutdown(ctx)
	<-done

	if sender.count() != 0 {
		t.Errorf("sender called %d times for a future email", sender.count())
	}
	if got := repo.snapshot(e.ID).Status; got != email.StatusQueued {
		t.Errorf("status = %s, want queued", got)
	}
}

// При превышении темпа письмо откладывается, а не отбрасывается: в очереди у
// него есть собственное время отправки.
func TestWorkerPostponesOnRateLimit(t *testing.T) {
	repo := newQueueRepo()
	sender := &scriptedSender{}

	e := repo.add(queuedEmail())

	cfg := Config{
		Count:        1,
		ClaimBatch:   10,
		PollInterval: 10 * time.Millisecond,
		LockTimeout:  time.Minute,
		Backoff:      []time.Duration{time.Millisecond},
	}
	pool := NewPool(cfg, repo, &recordingEvents{}, fixedLimiter{delay: time.Hour},
		func() (email.Sender, error) { return sender, nil }, logger.NewNop())

	runPool(t, pool, func() bool {
		s := repo.snapshot(e.ID)
		return s.Status == email.StatusQueued && s.ScheduledAt.After(time.Now().Add(time.Minute))
	})

	if sender.count() != 0 {
		t.Errorf("sender called %d times despite rate limit", sender.count())
	}

	stored := repo.snapshot(e.ID)
	if stored.Status != email.StatusQueued {
		t.Errorf("status = %s, want queued (postponed, not dropped)", stored.Status)
	}
	if !stored.ScheduledAt.After(time.Now().Add(30 * time.Minute)) {
		t.Errorf("scheduled_at = %v, want ~1h ahead", stored.ScheduledAt)
	}
}

// Суточный потолок останавливает отправку до вмешательства человека: без него
// цикл с ошибкой выберет квоту провайдера за минуты.
func TestWorkerHonoursDailyLimit(t *testing.T) {
	repo := newQueueRepo()
	repo.sentCount = 100
	sender := &scriptedSender{}

	repo.add(queuedEmail())
	pool := newPool(repo, &recordingEvents{}, sender, func(c *Config) {
		c.DailyLimit = 100
	})

	done := make(chan error, 1)
	go func() { done <- pool.Run() }()
	time.Sleep(150 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pool.Shutdown(ctx)
	<-done

	if sender.count() != 0 {
		t.Errorf("sender called %d times after the daily limit was reached", sender.count())
	}
}

func TestWorkerDailyLimitAllowsBelowThreshold(t *testing.T) {
	repo := newQueueRepo()
	repo.sentCount = 10
	sender := &scriptedSender{}

	e := repo.add(queuedEmail())
	pool := newPool(repo, &recordingEvents{}, sender, func(c *Config) {
		c.DailyLimit = 100
	})

	runPool(t, pool, func() bool {
		return repo.snapshot(e.ID).Status == email.StatusSent
	})

	if sender.count() != 1 {
		t.Errorf("sender called %d times, want 1", sender.count())
	}
}

// Несколько воркеров не должны отправить одно письмо дважды.
func TestWorkerPoolDoesNotDoubleSend(t *testing.T) {
	repo := newQueueRepo()
	sender := &scriptedSender{}

	const total = 20
	ids := make([]uuid.UUID, 0, total)
	for i := 0; i < total; i++ {
		ids = append(ids, repo.add(queuedEmail()).ID)
	}

	pool := newPool(repo, &recordingEvents{}, sender, func(c *Config) {
		c.Count = 4
		c.ClaimBatch = 3
	})

	runPool(t, pool, func() bool {
		for _, id := range ids {
			if repo.snapshot(id).Status != email.StatusSent {
				return false
			}
		}
		return true
	})

	if sender.count() != total {
		t.Errorf("sender called %d times for %d emails", sender.count(), total)
	}
}

// Ошибка чтения очереди не должна ронять воркера: БД может моргнуть.
func TestWorkerSurvivesClaimFailure(t *testing.T) {
	repo := newQueueRepo()
	repo.claimErr = errors.New("database is down")
	sender := &scriptedSender{}

	pool := newPool(repo, &recordingEvents{}, sender, nil)

	done := make(chan error, 1)
	go func() { done <- pool.Run() }()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := pool.Shutdown(ctx); err != nil {
		t.Errorf("pool did not stop cleanly after claim failures: %v", err)
	}
	<-done
}

// Воркер, не сумевший открыть соединение, не должен блокировать остановку.
func TestPoolStopsWhenSenderCannotBeCreated(t *testing.T) {
	repo := newQueueRepo()
	pool := NewPool(Config{Count: 2, PollInterval: 10 * time.Millisecond},
		repo, &recordingEvents{}, nil,
		func() (email.Sender, error) { return nil, errors.New("bad smtp config") },
		logger.NewNop())

	done := make(chan error, 1)
	go func() { done <- pool.Run() }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pool did not exit when senders could not be created")
	}
}
