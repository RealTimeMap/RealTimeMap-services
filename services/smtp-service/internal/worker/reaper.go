package worker

import (
	"context"
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/email"
	"go.uber.org/zap"
)

// Reaper возвращает в очередь письма, застрявшие в processing.
//
// Воркер помечает письмо processing перед отправкой и записывает исход после.
// Если он умрёт между этими шагами, письмо останется заблокированным навсегда:
// claim его больше не увидит, а никто не освободит.
//
// Есть шанс, что письмо всё-таки было сдано провайдеру и повторная отправка
// продублирует его у адресата. Выбор в пользу дубля сознательный: SMTP и
// Postgres не в одной транзакции, а неполученное письмо (код подтверждения,
// сброс пароля) хуже полученного дважды.
type Reaper struct {
	interval time.Duration
	emails   email.Repository
	logger   *zap.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func NewReaper(interval time.Duration, emails email.Repository, logger *zap.Logger) *Reaper {
	if interval <= 0 {
		interval = time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Reaper{
		interval: interval,
		emails:   emails,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
}

// Run запускает периодическую проверку и блокируется до остановки.
func (r *Reaper) Run() error {
	defer close(r.done)

	r.logger.Info("email reaper starting", zap.Duration("interval", r.interval))

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			r.logger.Info("email reaper stopped")
			return nil
		case <-ticker.C:
			r.sweep()
		}
	}
}

func (r *Reaper) Shutdown(ctx context.Context) error {
	r.cancel()

	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Reaper) sweep() {
	released, failed, err := r.emails.ReleaseExpired(r.ctx, time.Now())
	if err != nil {
		if r.ctx.Err() == nil {
			r.logger.Error("failed to release expired emails", zap.Error(err))
		}
		return
	}

	if released == 0 && failed == 0 {
		return
	}

	// Сообщение видно в логах: возврат писем означает, что воркер умер или
	// не уложился в LockTimeout — и то, и другое стоит заметить.
	r.logger.Warn("released stuck emails",
		zap.Int64("returned_to_queue", released),
		zap.Int64("marked_failed", failed),
	)
}
