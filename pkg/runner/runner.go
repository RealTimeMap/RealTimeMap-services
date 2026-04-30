package runner

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Server общий интерфейс для всех серверов, которыми управляет runner.
type Server interface {
	Run() error
	Shutdown(ctx context.Context) error
}

const defaultShutdownTimeout = 10 * time.Second

// Run запускает все переданные сервера и блокируется
// либо до первой ошибки в Run одного из серверов. После сигнала параллельно
// вызывает Shutdown у каждого сервера с общим таймаутом defaultShutdownTimeout.
func Run(logger *zap.Logger, servers ...Server) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	g, gCtx := errgroup.WithContext(ctx)

	for _, s := range servers {
		s := s
		g.Go(s.Run)
		g.Go(func() error {
			<-gCtx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
			defer cancel()
			return s.Shutdown(shutdownCtx)
		})
	}

	err := g.Wait()
	logger.Info("all servers stopped")
	return err
}
