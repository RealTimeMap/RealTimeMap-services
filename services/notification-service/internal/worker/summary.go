// Package worker досылает сводки по схлопнутым сериям уведомлений.
//
// Схлопывание в хендлере умеет только промолчать: оно решает, отправлять ли
// событие, в момент его прихода. Но последнее событие серии приходит, когда
// окно ещё идёт, — и без отдельного обхода накопленный счётчик так и остался
// бы в Redis, а пользователь не узнал бы о девяти пропущенных сообщениях.
//
// Обход реализует runner.Server и запускается рядом с HTTP и consumer.
package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/app/use_cases/notify"
	"github.com/RealTimeMap/RealTimeMap-backend/services/notification-service/internal/domain/collapse"
)

// Limiter — та часть collapse.Limiter, которая нужна обходу.
type Limiter interface {
	PendingKeys(ctx context.Context, now time.Time, limit int) ([]collapse.Key, error)
	Flush(ctx context.Context, key collapse.Key) (int, error)
}

// Notifier отправляет сводку.
type Notifier interface {
	HandleSummary(ctx context.Context, key collapse.Key, title, template string, count int) error
}

// summaryText — тексты сводок по классам уведомлений.
//
// Таблица здесь, а не в хендлере Kafka: обход восстанавливает серию из ключа
// в Redis и исходного события уже не видит — в ключе остаётся только Kind.
var summaryText = map[collapse.Kind]struct{ Title, Template string }{
	collapse.KindChatMessage: {Title: "Новые сообщения", Template: "У вас %d новых сообщений"},
	collapse.KindComment:     {Title: "Новые комментарии", Template: "%d новых комментариев к вашей метке"},
	collapse.KindSubscriber:  {Title: "Новые подписчики", Template: "У вас %d новых подписчиков"},
}

type Summary struct {
	limiter  Limiter
	notifier Notifier

	interval time.Duration
	batch    int

	logger *zap.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func NewSummary(
	limiter Limiter,
	notifier Notifier,
	interval time.Duration,
	batch int,
	logger *zap.Logger,
) *Summary {
	ctx, cancel := context.WithCancel(context.Background())
	return &Summary{
		limiter:  limiter,
		notifier: notifier,
		interval: interval,
		batch:    batch,
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
}

// Run блокируется до Shutdown.
func (s *Summary) Run() error {
	defer close(s.done)

	s.logger.Info("collapse summary worker starting",
		zap.Duration("interval", s.interval),
		zap.Int("batch", s.batch),
	)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("collapse summary worker stopped")
			return nil
		case <-ticker.C:
			s.tick(s.ctx)
		}
	}
}

func (s *Summary) Shutdown(ctx context.Context) error {
	s.cancel()
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// tick разбирает пачку закрытых серий.
//
// Ошибка на одной серии не останавливает проход: иначе один недоступный
// адресат заблокировал бы сводки всем остальным.
func (s *Summary) tick(ctx context.Context) {
	keys, err := s.limiter.PendingKeys(ctx, time.Now(), s.batch)
	if err != nil {
		s.logger.Warn("failed to fetch pending collapse keys", zap.Error(err))
		return
	}

	for _, key := range keys {
		if ctx.Err() != nil {
			return
		}

		count, err := s.limiter.Flush(ctx, key)
		if err != nil {
			s.logger.Warn("failed to flush collapse key",
				zap.String("collapse_key", key.String()),
				zap.Error(err),
			)
			continue
		}

		// Ноль означает, что серия закрылась ровно одним событием — оно уже
		// ушло пушем в момент открытия окна, и досылать нечего.
		if count < 1 {
			continue
		}

		text, ok := summaryText[key.Kind]
		if !ok {
			continue
		}

		if err := s.notifier.HandleSummary(ctx, key, text.Title, text.Template, count); err != nil {
			s.logger.Warn("failed to send collapse summary",
				zap.String("collapse_key", key.String()),
				zap.Int("count", count),
				zap.Error(err),
			)
			continue
		}

		s.logger.Debug("collapse summary sent",
			zap.String("collapse_key", key.String()),
			zap.Int("count", count),
		)
	}
}

// компиляция проверяет, что *notify.EventNotifyHandler подходит под Notifier.
var _ Notifier = (*notify.EventNotifyHandler)(nil)
