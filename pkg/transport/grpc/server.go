package servergrpc

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Registrar func(*grpc.Server)

type Config struct {
	Port int `yaml:"port" env:"GRPC_PORT"`
}

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	logger     *zap.Logger
}

func NewServer(cfg Config, logger *zap.Logger, registrars ...Registrar) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", cfg.Port, err)
	}

	s := grpc.NewServer()
	for _, r := range registrars {
		r(s)
	}
	reflection.Register(s)
	return &Server{
		grpcServer: s,
		logger:     logger,
		listener:   listener,
	}, nil
}

func (s *Server) Run() error {
	s.logger.Info("gRPC server starting", zap.String("address", s.listener.Addr().String()))
	return s.grpcServer.Serve(s.listener)
}

// Shutdown пытается graceful-остановить сервер.
// Если ctx истечёт раньше — делает force-stop и возвращает ctx.Err().
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("gRPC server stopping")
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.grpcServer.Stop()
		return ctx.Err()
	}
}
