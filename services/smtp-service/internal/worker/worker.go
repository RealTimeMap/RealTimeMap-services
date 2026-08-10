// Package worker разбирает очередь писем и отправляет их.
package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/utils"
	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Config — параметры пула.
type Config struct {
	Count        int
	ClaimBatch   int
	PollInterval time.Duration
	LockTimeout  time.Duration
	Backoff      []time.Duration

	// DailyLimit — предохранитель от разгона при баге. 0 отключает проверку.
	DailyLimit int
}

// RateLimiter откладывает отправку, когда домен получателя перегружен.
type RateLimiter interface {
	// Reserve возвращает, сколько нужно подождать перед отправкой на домен.
	Reserve(domain string) time.Duration
}

// Pool — группа воркеров, разбирающих очередь.
type Pool struct {
	cfg       Config
	emails    email.Repository
	events    email.EventRepository
	limiter   RateLimiter
	newSender func() (email.Sender, error)
	logger    *zap.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// limitReached запоминает, что суточный потолок уже отмечен в логе:
	// иначе каждый цикл опроса писал бы одно и то же предупреждение.
	limitMu      sync.Mutex
	limitReached bool
}

func NewPool(
	cfg Config,
	emails email.Repository,
	events email.EventRepository,
	limiter RateLimiter,
	newSender func() (email.Sender, error),
	logger *zap.Logger,
) *Pool {
	if cfg.Count <= 0 {
		cfg.Count = 3
	}
	if cfg.ClaimBatch <= 0 {
		cfg.ClaimBatch = 10
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.LockTimeout <= 0 {
		cfg.LockTimeout = 2 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		cfg:       cfg,
		emails:    emails,
		events:    events,
		limiter:   limiter,
		newSender: newSender,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
	}
}

// Run запускает воркеров и блокируется до остановки.
func (p *Pool) Run() error {
	defer close(p.done)

	p.logger.Info("email workers starting",
		zap.Int("count", p.cfg.Count),
		zap.Int("claim_batch", p.cfg.ClaimBatch),
	)

	var wg sync.WaitGroup
	for i := 0; i < p.cfg.Count; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			p.work(fmt.Sprintf("worker-%d-%s", n, uuid.New().String()[:8]))
		}(i)
	}
	wg.Wait()

	p.logger.Info("email workers stopped")
	return nil
}

// Shutdown сигналит воркерам завершиться и ждёт, пока они дорабатывают
// текущие письма.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.logger.Info("email workers stopping")
	p.cancel()

	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		// Воркер не успел закрыть текущую отправку: письмо останется в
		// processing, и его подберёт reaper после LockTimeout.
		return ctx.Err()
	}
}

// work — цикл одного воркера
func (p *Pool) work(workerID string) {
	log := p.logger.With(zap.String("worker_id", workerID))

	sender, err := p.newSender()
	if err != nil {
		log.Error("failed to create sender, worker will not start", zap.Error(err))
		return
	}
	defer closeSender(sender, log)

	for {
		if p.ctx.Err() != nil {
			return
		}

		processed, err := p.processBatch(workerID, sender, log)
		if err != nil {
			log.Error("batch processing failed", zap.Error(err))
		}

		// Пауза только когда работы не было: иначе непрерывный поток писем
		// обрабатывается без искусственных задержек.
		if processed == 0 {
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(p.cfg.PollInterval):
			}
		}
	}
}

func (p *Pool) processBatch(workerID string, sender email.Sender, log *zap.Logger) (int, error) {
	if p.dailyLimitReached() {
		return 0, nil
	}

	batch, err := p.emails.Claim(p.ctx, workerID, p.cfg.ClaimBatch, p.cfg.LockTimeout)
	if err != nil {
		if p.ctx.Err() != nil {
			return 0, nil
		}
		return 0, fmt.Errorf("claim: %w", err)
	}

	for i := range batch {
		// Остановку проверяем перед каждым письмом, а не только между
		// батчами: батч из десяти писем отправляется несколько секунд.
		if p.ctx.Err() != nil {
			p.releaseUnsent(batch[i:], "released on shutdown", log)
			return len(batch), nil
		}

		p.deliver(&batch[i], sender, log)
	}

	return len(batch), nil
}

// deliver отправляет одно письмо и записывает исход.
func (p *Pool) deliver(e *email.Email, sender email.Sender, log *zap.Logger) {
	log = log.With(
		zap.String("email_id", e.ID.String()),
		zap.String("template", e.TemplateID),
		zap.String("to", email.MaskEmail(e.ToEmail)),
		zap.Uint("attempt", e.Attempt),
		zap.String("trace_id", e.TraceID),
	)

	// Проверка после claim, а не только до него: между решением забрать батч и
	// отправкой конкретного письма лимит мог быть исчерпан — параллельным
	// воркером или предыдущими письмами этого же батча. Письмо возвращается в
	// очередь, иначе висело бы в processing до срабатывания reaper.
	if p.dailyLimitReached() {
		p.reschedule(e, time.Now().Add(p.cfg.PollInterval), "daily send limit reached", log)
		return
	}

	if delay := p.rateLimitDelay(e.ToEmail); delay > 0 {
		// Письмо откладывается, а не отбрасывается: в очереди у него есть
		// собственное время отправки.
		p.reschedule(e, time.Now().Add(delay), "rate limited", log)
		log.Debug("email postponed by rate limit", zap.Duration("delay", delay))
		return
	}

	start := time.Now()
	err := sender.Send(p.ctx, email.OutgoingMessage{
		To:      e.ToEmail,
		Subject: e.Subject,
		HTML:    e.HTML,
	})
	elapsed := time.Since(start)

	switch {
	case err == nil:
		p.markSent(e, elapsed, log)

	case errors.Is(err, email.ErrPermanent):
		p.markFailed(e, err, elapsed, log)

	case p.ctx.Err() != nil && errors.Is(err, email.ErrTransient):
		// Сервис останавливается — обрыв на этом фоне не считаем попыткой
		// и возвращаем письмо в очередь немедленно.
		p.reschedule(e, time.Now(), "worker shutdown", log)

	case e.Attempt >= e.MaxAttempt:
		p.markFailed(e, fmt.Errorf("%w (attempts exhausted)", err), elapsed, log)

	default:
		delay := backoff(p.cfg.Backoff, e.Attempt)
		p.reschedule(e, time.Now().Add(delay), err.Error(), log)
		log.Warn("email delivery failed, will retry",
			zap.Duration("retry_in", delay),
			zap.Duration("duration", elapsed),
			zap.Error(err),
		)
	}
}

func (p *Pool) markSent(e *email.Email, elapsed time.Duration, log *zap.Logger) {
	// Контекст фоновый: письмо уже ушло, и статус обязан записаться даже при
	// остановке сервиса — иначе reaper вернёт его в очередь и адресат получит
	// письмо второй раз.
	ctx, cancel := detachedContext()
	defer cancel()

	now := time.Now()
	if err := p.emails.MarkSent(ctx, e.ID, now); err != nil {
		log.Error("email sent but status not saved, may be delivered twice", zap.Error(err))
		return
	}

	e.Status = email.StatusSent
	e.SentAt = &now
	p.appendEvent(ctx, e, email.EventSent, map[string]any{
		"duration_ms": elapsed.Milliseconds(),
	})

	log.Info("email sent", zap.Int64("duration_ms", elapsed.Milliseconds()))
}

func (p *Pool) markFailed(e *email.Email, cause error, elapsed time.Duration, log *zap.Logger) {
	ctx, cancel := detachedContext()
	defer cancel()

	if err := p.emails.MarkFailed(ctx, e.ID, cause.Error()); err != nil {
		log.Error("failed to mark email as failed", zap.Error(err))
		return
	}

	e.Status = email.StatusFailed
	p.appendEvent(ctx, e, email.EventFailed, map[string]any{
		"error":       cause.Error(),
		"duration_ms": elapsed.Milliseconds(),
	})

	log.Error("email delivery failed permanently",
		zap.Int64("duration_ms", elapsed.Milliseconds()),
		zap.Error(cause),
	)
}

func (p *Pool) reschedule(e *email.Email, at time.Time, reason string, log *zap.Logger) {
	ctx, cancel := detachedContext()
	defer cancel()

	if err := p.emails.Reschedule(ctx, e.ID, at, reason); err != nil {
		log.Error("failed to reschedule email", zap.Error(err))
		return
	}

	e.Status = email.StatusQueued
	e.ScheduledAt = at
	p.appendEvent(ctx, e, email.EventRetried, map[string]any{
		"reason":       reason,
		"scheduled_at": at.Format(time.RFC3339),
	})
}

// releaseUnsent возвращает в очередь письма, которые воркер забрал, но не
// успел отправить до остановки. Без этого они ждали бы LockTimeout.
func (p *Pool) releaseUnsent(batch []email.Email, reason string, log *zap.Logger) {
	if len(batch) == 0 {
		return
	}

	ctx, cancel := detachedContext()
	defer cancel()

	now := time.Now()
	for i := range batch {
		if err := p.emails.Reschedule(ctx, batch[i].ID, now, reason); err != nil {
			log.Warn("failed to release claimed email",
				zap.String("email_id", batch[i].ID.String()),
				zap.String("reason", reason),
				zap.Error(err),
			)
		}
	}

	log.Info("released claimed emails",
		zap.String("reason", reason),
		zap.Int("count", len(batch)),
	)
}

func (p *Pool) rateLimitDelay(address string) time.Duration {
	if p.limiter == nil {
		return 0
	}

	domain, err := utils.ExtractEmailDomain(address)
	if err != nil {
		// Адрес проверен при постановке в очередь; сюда попасть не должны.
		return 0
	}

	return p.limiter.Reserve(domain)
}

// dailyLimitReached защищает от разгона при баге: без него цикл с ошибкой
// может выбрать суточную квоту провайдера за минуты и привести к блокировке
// ящика.
func (p *Pool) dailyLimitReached() bool {
	if p.cfg.DailyLimit <= 0 {
		return false
	}

	sent, err := p.emails.CountSentSince(p.ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		p.logger.Warn("daily limit check failed", zap.Error(err))
		return false
	}

	if sent < int64(p.cfg.DailyLimit) {
		p.limitMu.Lock()
		p.limitReached = false
		p.limitMu.Unlock()
		return false
	}

	p.limitMu.Lock()
	first := !p.limitReached
	p.limitReached = true
	p.limitMu.Unlock()

	if first {
		p.logger.Warn("daily send limit reached, pausing delivery",
			zap.Int64("sent_24h", sent),
			zap.Int("limit", p.cfg.DailyLimit),
		)
	}

	return true
}

func (p *Pool) appendEvent(ctx context.Context, e *email.Email, eventType email.EventType, details map[string]any) {
	event := &email.Event{
		EmailID:         e.ID,
		EventType:       eventType,
		WorkerID:        e.WorkerID,
		OccurredTime:    time.Now(),
		TemplateVersion: e.TemplateVersion,
		Details:         details,
		Attempt:         e.Attempt,
		MaxAttempt:      e.MaxAttempt,
	}

	if err := p.events.Append(ctx, event); err != nil {
		p.logger.Warn("failed to append email event",
			zap.String("email_id", e.ID.String()),
			zap.String("event_type", string(eventType)),
			zap.Error(err),
		)
	}
}

func closeSender(sender email.Sender, log *zap.Logger) {
	closer, ok := sender.(interface{ Close() error })
	if !ok {
		return
	}
	if err := closer.Close(); err != nil {
		log.Debug("failed to close sender", zap.Error(err))
	}
}

// detachedContext даёт короткий контекст, не связанный с остановкой сервиса:
// исход отправки нужно записать в любом случае.
func detachedContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}
