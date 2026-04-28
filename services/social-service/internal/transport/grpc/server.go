package grpc

import (
	"fmt"
	"net"

	pb "github.com/RealTimeMap/RealTimeMap-backend/pkg/pb/profile"
	profilegrpc "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/grpc/profile"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	logger     *zap.Logger
}

func NewServer(profileHandler *profilegrpc.Handler, port int, logger *zap.Logger) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterProfileServiceServer(grpcServer, profileHandler)
	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		listener:   listener,
		logger:     logger,
	}, nil
}

func (s *Server) Run() error {
	s.logger.Info("gRPC server starting", zap.String("address", s.listener.Addr().String()))
	return s.grpcServer.Serve(s.listener)
}

func (s *Server) Stop() {
	s.logger.Info("gRPC server stopping")
	s.grpcServer.GracefulStop()
}
